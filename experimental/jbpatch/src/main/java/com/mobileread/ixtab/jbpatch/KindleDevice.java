package com.mobileread.ixtab.jbpatch;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.IOException;
import java.util.Map;
import java.util.TreeMap;

public final class KindleDevice {

  /* The order of the declarations in this file is somewhat strange.
   * This is due to the order that classes are initialized. In other
   * words: if you change the order here, you might get weird exceptions.
   */
  private static final String VERSION_FILENAME = "/etc/version.txt";

  private static final String MD5_KT_510_1557760049 = "4d14b7af911080e6a073121eb3262e54";
  private static final String MD5_KT_511_1568190003 = "27828f5c87c8d7f345ddf157bdcac1df";

  private static String getVersionMd5() {
    /* The Amazon logic to determine the version seems to use the native
     * libraries liblab126utilsjni and liblab126utils, to determine the
     * "revision" of the Kindle from /etc/version.txt. Without having
     * thoroughly decompiled everything, the device info
     * "Kindle 5.1.0 (1557760049)" seems to stem from /etc/prettyversion.txt
     * (for the first part), and the first line of /etc/version.txt:
     * In this case it was:
     * System Software Version: 049-juno_2_yoshi-155776
     * So the "end" (155776), concatenated with the "beginning" (049), and
     * some filler zeroes in between. We may need to actually parse this line
     * at some later point in time, but for now, this method simply returns
     * the string value of the MD5 hash of the first line in /etc/version.txt,
     * or null on error.
     */

    String md5 = null;
    BufferedReader reader = null;
    try {
      reader = new BufferedReader(new FileReader(VERSION_FILENAME));
      String line = reader.readLine();
      if (line != null) {
        md5 = MD5.getMd5String(line.getBytes());
      }
    } catch (IOException e) {
    } finally {
      if (reader != null) {
        try {
          reader.close();
        } catch (IOException e) {
        }
        ;
      }
    }
    return md5;
  }

  private static Map<String, String> buildFirmwareMap() {
    Map<String, String> map = new TreeMap<>();
    map.put(MD5_KT_510_1557760049, "Kindle 5.1.0 (1557760049)");
    map.put(MD5_KT_511_1568190003, "Kindle 5.1.1 (1568190003)");
    return map;
  }

  private static final Map md5Map = buildFirmwareMap();

  public static KindleDevice THIS_DEVICE = new KindleDevice(getVersionMd5());

  public static final KindleDevice KT_510_1557760049 = new KindleDevice(MD5_KT_510_1557760049);
  public static final KindleDevice KT_511_1568190003 = new KindleDevice(MD5_KT_511_1568190003);

  public final String md5;
  private final String description;

  private KindleDevice(String md5) {
    this.md5 = md5;
    this.description = (String) md5Map.get(md5);
  }

  public String getDescription() {
    return description;
  }

  public String getSafeDescription() {
    return description != null ? description : "UNKNOWN FIRMWARE WITH ID " + md5;
  }

  public String toString() {
    return getSafeDescription();
  }

  public boolean equals(Object obj) {
    if (this == obj) return true;
    if (obj == null) return false;
    if (getClass() != obj.getClass()) return false;
    KindleDevice other = (KindleDevice) obj;
    if (md5 == null) {
      if (other.md5 != null) return false;
    } else if (!md5.equals(other.md5)) return false;
    return true;
  }

  public int hashCode() {
    return md5.hashCode();
  }
}
