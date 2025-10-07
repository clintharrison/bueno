package com.mobileread.ixtab.jbpatch;

public final class Environment {
  private Environment() {}

  public static boolean isKindle() {
    // trivial check, but should work
    return System.getProperty("os.arch", "unknown").toLowerCase().startsWith("arm");
  }

  private static String firmware = null;

  /**
   * Returns the Kindle Firmware version as a String. <b>NOTE:</b> This does not necessarily cover
   * all available versions, and it is <b>not</b> the same as the Device Information dialog. For
   * instance, Kindle Touch versions before 5.1.0 are not "detected", and 5.3.2.1 and 5.3.2 are
   * returned as the same version (5.3.2).
   *
   * <p>In other words: this method uses a "best effort" method to determine the currently running
   * firmware, by examining the classes available in the framework.
   *
   * <p>Note that this will normally not interfere with the correct functionality of individual
   * patches, because a) this is an optional method that patches can, but are not required to,
   * invoke (generally in the {@link Patch#isAvailable()} method), and b) patches must still
   * register for the exact MD5 sum of the classes to patch. So in the worst case, a patch might not
   * be available/executed even if it could.
   *
   * @return the Kindle Firmware version, e.g., "5.1.0" or "5.3.4"
   */
  public static String getFirmware() {
    if (firmware == null) {
      synchronized (Environment.class) {
        if (firmware == null) {
          firmware = "5.1.0";
          try {
            Class.forName("com.amazon.ebook.util.lang.UUID");
          } catch (Throwable t) {
            firmware = "5.3.1";
            try {
              // present in 5.3.2 (Touch), but not 5.3.1
              Class.forName("com.amazon.kindle.booklet.ad.resources.AdResources_sq");
              firmware = "5.3.2";
            } catch (Throwable t2) {
              // 5.3.3 (PW).
              try {
                Class.forName("com.amazon.ebook.booklet.topazreader.impl.A");
                firmware = "5.3.3";
                try {
                  Class.forName("com.amazon.ebook.booklet.reader.sdk.internal.SimpleBookView");
                  firmware = "5.3.4";
                } catch (Throwable t4) {
                }
              } catch (Throwable t3) {
              }
            }
          }
        }
      }
    }
    return firmware;
  }

  /**
   * Returns the release date of the installed JBPatch version in YYYYMMDD format. This should make
   * it easy to depend on "version X or newer", regardless of the human-readable JBPatch version.
   *
   * @return
   */
  public static int getJBPatchVersionDate() {
    return JBPatchMetadata.DATE;
  }
}
