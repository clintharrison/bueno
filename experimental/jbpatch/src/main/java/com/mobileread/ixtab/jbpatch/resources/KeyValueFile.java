package com.mobileread.ixtab.jbpatch.resources;

import com.mobileread.ixtab.jbpatch.Log;
import java.io.BufferedReader;
import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;
import java.util.StringTokenizer;

public class KeyValueFile implements KeyValueResource {

  public static int FLAG_NONE = 0;
  public static int FLAG_DYNAMIC = 1;
  public static int FLAG_WRITABLE = 2;

  private String[] keys;
  private String[] values;
  private final File primaryFile;
  private final int flags;

  private long lastTimestamp = 0;

  public KeyValueFile(int flags, File primary) {
    this.primaryFile = primary;
    this.flags = flags;
    reload();
  }

  public String getValue(String key) {
    if (isDynamic()) {
      if (primaryFile.lastModified() != lastTimestamp) {
        reload();
      }
    }
    int index = Arrays.binarySearch(keys, key);
    return index < 0 ? null : values[index];
  }

  private boolean reload() {
    BufferedReader r = null;
    List keyList = new ArrayList();
    List valueList = new ArrayList();
    boolean ok = false;
    try {
      if (primaryFile.exists()) {
        lastTimestamp = primaryFile.lastModified();
        r = new BufferedReader(new InputStreamReader(new FileInputStream(primaryFile), "UTF-8"));
        initialize(keyList, valueList, r, primaryFile.getAbsolutePath());
        ok = true;
      }
    } catch (IOException e) {
      keyList.clear();
      valueList.clear();
      e.printStackTrace(Log.INSTANCE);
    } finally {
      if (r != null) {
        try {
          r.close();
        } catch (IOException e) {
        }
      }
    }
    keys = new String[keyList.size()];
    keyList.toArray(keys);
    values = new String[valueList.size()];
    valueList.toArray(values);
    return ok;
  }

  private void initialize(List keys, List values, BufferedReader reader, String filename)
      throws IOException {
    int ln = 0;
    for (String line = reader.readLine(); line != null; line = reader.readLine()) {
      ++ln;
      line = line.trim();
      if (line.length() == 0 || line.startsWith("#")) {
        continue;
      }
      int eq = line.indexOf('=');
      if (eq == -1) {
        Log.INSTANCE.println("W: Invalid line #" + ln + " in file " + filename + ": missing =");
        continue;
      }
      String k = line.substring(0, eq);
      String v = unescape(line.substring(eq + 1));

      int pos = Collections.binarySearch(keys, k);
      if (pos < 0) {
        pos = -(pos + 1);
      }
      keys.add(pos, k);
      values.add(pos, v);
    }
  }

  public boolean setValue(String key, String value) {
    return setValue(key, value, true);
  }

  public boolean setValue(String key, String value, boolean autoCommit) {
    if (!isWritable()) {
      throw new IllegalStateException(primaryFile + " was opened in readonly mode");
    }
    if (value == null) {
      return false;
    }
    boolean dirty = false;
    int pos = Arrays.binarySearch(keys, key);
    if (pos < 0) {
      // this should not normally happen, but you never know...
      dirty = true;
      pos = -(pos + 1);
      String[] nKeys = new String[keys.length + 1];
      String[] nVals = new String[nKeys.length];
      nKeys[pos] = key;
      nVals[pos] = value;
      System.arraycopy(keys, 0, nKeys, 0, pos);
      System.arraycopy(values, 0, nVals, 0, pos);
      System.arraycopy(keys, pos, nKeys, pos + 1, keys.length - pos);
      System.arraycopy(values, pos, nVals, pos + 1, values.length - pos);
      keys = nKeys;
      values = nVals;
    } else {
      if (!values[pos].equals(value)) {
        dirty = true;
        values[pos] = value;
      }
    }
    if (!dirty) {
      return true;
    }
    if (autoCommit) {
      return commitChanges();
    } else {
      return true;
    }
  }

  public boolean remove(String key) {
    return remove(key, true);
  }

  public boolean remove(String key, boolean autoCommit) {
    if (!isWritable()) {
      throw new IllegalStateException(primaryFile + " was opened in readonly mode");
    }
    int pos = Arrays.binarySearch(keys, key);
    if (pos < 0) return true;
    String[] nKeys = new String[keys.length - 1];
    String[] nVals = new String[nKeys.length];
    System.arraycopy(keys, 0, nKeys, 0, pos);
    System.arraycopy(values, 0, nVals, 0, pos);
    System.arraycopy(keys, pos + 1, nKeys, pos, nKeys.length - pos);
    System.arraycopy(values, pos + 1, nVals, pos, nVals.length - pos);
    keys = nKeys;
    values = nVals;

    if (autoCommit) {
      return commitChanges();
    }
    return true;
  }

  public boolean commitChanges() {
    return write(primaryFile);
  }

  private boolean write(File file) {
    if (file == null) {
      return true;
    }
    try {
      FileOutputStream fos = new FileOutputStream(file);
      PrintStream ps = new PrintStream(fos, false, "UTF-8");
      ps.print("# DO NOT DELETE THIS FILE.\r\n\r\n");
      ps.print("# This file was automatically generated or updated.\r\n");
      ps.print(
          "# While is recommended to use the jbpatch UI to modify the values contained hereafter,"
              + " \r\n");
      ps.print(
          "# you can also manually change them. Just be aware that you may need to restart\r\n");
      ps.print("# your Kindle before you see an effect.\r\n\r\n");
      for (int i = 0; i < keys.length; ++i) {
        ps.print(keys[i]);
        ps.print("=");
        ps.print(escape(values[i]));
        ps.print("\r\n");
      }
      ps.flush();
      ps.close();
      lastTimestamp = file.lastModified();
    } catch (IOException io) {
      Log.INSTANCE.println("Error while writing to " + file + ":");
      io.printStackTrace(Log.INSTANCE);
      return false;
    }
    return true;
  }

  static String escape(String in) {
    StringTokenizer tokens = new StringTokenizer(in, "\n");
    int count = tokens.countTokens();
    if (count == 1) {
      return in;
    }
    int current = 1;
    StringBuffer out = new StringBuffer();
    while (tokens.hasMoreTokens()) {
      out.append(tokens.nextToken());
      if (current++ < count) {
        out.append("\\n");
      }
    }
    return out.toString();
  }

  private String unescape(String in) {
    int pos = 0;
    StringBuffer out = new StringBuffer();
    for (int nl = in.indexOf("\\n", pos); nl >= 0; nl = in.indexOf("\\n", pos)) {
      String chunk = in.substring(pos, nl);
      out.append(chunk);
      out.append("\n");
      pos = nl + 2;
    }
    out.append(in.substring(pos));
    return out.toString();
  }

  private boolean isWritable() {
    return (flags & FLAG_WRITABLE) == FLAG_WRITABLE;
  }

  private boolean isDynamic() {
    return (flags & FLAG_DYNAMIC) == FLAG_DYNAMIC;
  }

  public List listKeys() {
    return Arrays.asList(keys);
  }
}
