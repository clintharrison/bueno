package com.mobileread.ixtab.jbpatch;

import com.mobileread.ixtab.jbpatch.resources.KeyValueFile;
import java.io.File;
import java.io.FilenameFilter;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;

public class PatchRepository {

  public static final String EXTENSION_PATCH_STANDALONE = ".class";
  public static final String EXTENSION_PATCH_JARRED = ".jar";
  private static final String EXTENSION_TXT = ".txt";

  private static final String CONFIGFILE_NAME = "CONFIG.txt";
  private static final char[] SPECIAL_CHARS_OK = new char[] {'.', '_'};

  private static final String VALUE_ENABLED = "enabled";
  private static final String VALUE_DISABLED = "disabled";

  private static final PatchRepository instance = new PatchRepository();

  private final KeyValueFile backend;
  private final Map patchStateMap = new TreeMap();

  public static PatchRepository getInstance() {
    return instance;
  }

  private PatchRepository() {
    File primary = new File(KindleDirectories.LOCAL_DIRECTORY + "/" + CONFIGFILE_NAME);
    backend = new KeyValueFile(KeyValueFile.FLAG_WRITABLE, primary);
  }

  Map initialize() {
    boolean updateBackend = false;
    File[] files = listPresentPatchFiles();
    List keysInBackend = new ArrayList(backend.listKeys());
    Map patchStates = new TreeMap();
    for (int i = 0; i < files.length; ++i) {
      Boolean active = null;
      String id = getPatchId(files[i].getName());
      keysInBackend.remove(id);
      String status = backend.getValue(id);
      if (status == null) {
        log("I: " + id + " was not found in configuration, adding entry and enabling patch");
        if (!backend.setValue(id, VALUE_ENABLED, false)) {
          log("E: failed to add patch " + id + " to configuration!");
        }
        active = Boolean.TRUE;
        updateBackend = true;
      } else {
        active = parseStatus(status);
        if (active == null) {
          log(
              "W: "
                  + id
                  + " has wrong entry in configuration file, fixing entry and enabling patch");
          if (!backend.setValue(id, VALUE_ENABLED, false)) {
            log("E: failed to fix state for patch " + id + " in configuration!");
          }
          active = Boolean.TRUE;
          updateBackend = true;
        }
      }
      patchStates.put(files[i], active);
    }
    for (int i = 0; i < keysInBackend.size(); ++i) {
      String id = (String) keysInBackend.get(i);
      if (!backend.remove(id, false)) {
        log("E: failed to remove patch " + id + " from configuration!");
      }
      updateBackend = true;
    }
    if (updateBackend) {
      if (!backend.commitChanges()) {
        log("E: failed to update configuration file!");
      } else {
        log("I: updated configuration file");
      }
    }
    return patchStates;
  }

  private File[] listPresentPatchFiles() {
    File dir = new File(KindleDirectories.LOCAL_DIRECTORY);
    if (!dir.exists() || !dir.isDirectory()) {
      return new File[0];
    }
    return dir.listFiles(new FilenamesFilter(false, false));
  }

  private void log(String msg) {
    Log.INSTANCE.println(msg);
  }

  private Boolean parseStatus(String text) {
    if (VALUE_ENABLED.equals(text)) {
      return Boolean.TRUE;
    }
    if (VALUE_DISABLED.equals(text)) {
      return Boolean.FALSE;
    }
    return null;
  }

  private String getPatchId(String filename) {
    if (filename.endsWith(EXTENSION_PATCH_STANDALONE)) {
      return filename.substring(0, filename.length() - EXTENSION_PATCH_STANDALONE.length());
    } else if (filename.endsWith(EXTENSION_PATCH_JARRED)) {
      return filename.substring(0, filename.length() - EXTENSION_PATCH_JARRED.length());
    }
    throw new IllegalStateException();
  }

  void addAvailable(Patch patch, Boolean active) {
    patchStateMap.put(patch, active);
  }

  @SuppressWarnings("removal")
  public static Character checkForWrongCharacters(String name) {
    // no regex support, so do it manually
    char[] chars = name.toCharArray();
    for (int i = 0; i < chars.length; ++i) {
      char c = chars[i];
      if (c >= '0' && c <= '9') continue;
      if (c >= 'a' && c <= 'z') continue;
      if (c >= 'A' && c <= 'Z') continue;
      if (isAllowedSpecialChar(c)) continue;
      return new Character(c);
    }
    return null;
  }

  private static boolean isAllowedSpecialChar(char c) {
    for (int i = 0; i < SPECIAL_CHARS_OK.length; ++i) {
      if (c == SPECIAL_CHARS_OK[i]) return true;
    }
    return false;
  }

  public static class FilenamesFilter implements FilenameFilter {

    private final boolean allowAuxiliaryFiles;
    private final boolean allowMainConfigurationFile;

    public FilenamesFilter(boolean allowAuxiliaryFiles, boolean allowMainConfigurationFile) {
      super();
      this.allowAuxiliaryFiles = allowAuxiliaryFiles;
      this.allowMainConfigurationFile = allowMainConfigurationFile;
    }

    public boolean accept(File dir, String name) {
      if (name.equals(CONFIGFILE_NAME)) {
        return allowMainConfigurationFile;
      }
      if (allowAuxiliaryFiles && name.endsWith(EXTENSION_TXT)) {
        return true;
      }
      if (name.endsWith(EXTENSION_PATCH_STANDALONE) || name.endsWith(EXTENSION_PATCH_JARRED)) {
        if (checkForWrongCharacters(name) == null) {
          return true;
        }
      }
      return false;
    }
  }

  public synchronized Map getAvailablePatches() {
    return patchStateMap;
  }
  ;

  public boolean setPatchState(Patch patch, boolean enabled) {
    patchStateMap.put(patch, Boolean.valueOf(enabled));
    return backend.setValue(patch.id(), enabled ? VALUE_ENABLED : VALUE_DISABLED, true);
  }
}
