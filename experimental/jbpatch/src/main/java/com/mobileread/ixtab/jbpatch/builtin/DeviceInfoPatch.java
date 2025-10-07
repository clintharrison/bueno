package com.mobileread.ixtab.jbpatch.builtin;

import com.mobileread.ixtab.jbpatch.JBPatchMetadata;
import com.mobileread.ixtab.jbpatch.Patch;
import com.mobileread.ixtab.jbpatch.PatchMetadata;
import com.mobileread.ixtab.jbpatch.PatchMetadata.PatchableClass;
import java.text.MessageFormat;
import java.util.Map;
import serp.bytecode.BCClass;
import serp.bytecode.Code;

public class DeviceInfoPatch extends Patch {
  private static final String FW510_CLASS = "com.amazon.kindle.settings.dialog.DeviceInfoDialog";
  private static final String FW510_MD5_IN = "9f393118b394eaa5ffcca7f44e47db2b";
  private static final String FW510_MD5_OUT = "d12a6b3fdcd14d39a8beb9cd4436e6cc";

  private static final String FW53X_CLASS = "com.amazon.kindle.settings.e.X";
  private static final String FW531_MD5_IN = "5a65cdf8740e184fc18e8e4a31d0167a";
  private static final String FW531_MD5_OUT = "2c8d4c9a4f0ef88ae85f186fcc38790a";

  private static final String FW532_MD5_IN = "36008656fe97d502c71b028c47901ac9";
  private static final String FW532_MD5_OUT = "d59a6e82d2f395350948a341ae0df688";

  private static final String FW533_MD5_IN = "761e84ba4c259c4716f43f785a67df83";
  private static final String FW533_MD5_OUT = "35ee6a5ccb5e80e65fd3df3d1d779380";

  private static final String FW534_MD5_IN = "07b2488eb9000de2d5bd413a6087daaf";
  private static final String FW534_MD5_OUT = "26682a4a28e0a8e73196475ec290b7b5";

  public int getVersion() {
    return 1;
  }

  public boolean isAvailable() {
    return true;
  }

  public PatchMetadata getMetadata() {
    PatchableClass c510 =
        new PatchableClass(FW510_CLASS).withChecksums(FW510_MD5_IN, FW510_MD5_OUT);
    PatchableClass c53x =
        new PatchableClass(FW53X_CLASS)
            .withChecksums(FW531_MD5_IN, FW531_MD5_OUT)
            .withChecksums(FW532_MD5_IN, FW532_MD5_OUT)
            .withChecksums(FW533_MD5_IN, FW533_MD5_OUT)
            .withChecksums(FW534_MD5_IN, FW534_MD5_OUT);
    return new PatchMetadata(this).withClass(c510).withClass(c53x);
  }

  public String perform(String md5, BCClass clazz) throws Throwable {
    if (md5.equals(FW510_MD5_IN)) {
      return patchFw510(clazz);
    } else if (md5.equals(FW531_MD5_IN)) {
      return patchFw53X(clazz, "WkA");
    } else if (md5.equals(FW532_MD5_IN)) {
      return patchFw53X(clazz, "VJA");
    } else if (md5.equals(FW533_MD5_IN)) {
      return patchFw53X(clazz, "kkA");
    } else if (md5.equals(FW534_MD5_IN)) {
      return patchFw53X(clazz, "ZJA");
    } else {
      return "Unsupported MD5: " + md5;
    }
  }

  private String patchFw510(BCClass clazz) throws NoSuchMethodException {
    Code c = clazz.getDeclaredMethods("h")[0].getCode(false);

    // substitute call to MessageFormat.format by call to ourselves
    c.before(52);
    c.invokestatic()
        .setMethod(
            DeviceInfoPatch.class.getMethod("format", new Class[] {String.class, Object[].class}));
    c.next();
    c.remove();

    // increase the dialog height to allow for the added line
    c.before(109);
    c.constant().setValue(20);
    c.iadd();
    return null;
  }

  private String patchFw53X(BCClass clazz, String method) throws Exception {
    Code c = clazz.getDeclaredMethod(method).getCode(false);

    // substitute call to MessageFormat.format by call to ourselves
    c.before(79);
    c.invokestatic()
        .setMethod(
            DeviceInfoPatch.class.getMethod("format", new Class[] {String.class, Object[].class}));
    c.next();
    c.remove();

    // increase the dialog height to allow for the added line
    c.before(186);
    c.constant().setValue(20);
    c.iadd();
    return null;
  }

  public static String format(String pattern, Object[] arguments) {
    if (pattern.endsWith("</html>")) {
      pattern = pattern.substring(0, pattern.length() - 7);
      pattern += "<br/><b>JBPatch:</b>&nbsp;" + JBPatchMetadata.VERSION + "</html>";
    }
    return MessageFormat.format(pattern, arguments);
  }

  protected void initLocalization(String locale, Map map) {
    // because this is a very special case, we can indeed afford to do
    // nothing here.
  }
}
