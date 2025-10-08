package com.mobileread.ixtab.patch;

import com.amazon.ebook.booklet.reader.resources.ReaderResources;
import com.mobileread.ixtab.jbpatch.Environment;
import com.mobileread.ixtab.jbpatch.Patch;
import com.mobileread.ixtab.jbpatch.PatchMetadata;
import com.mobileread.ixtab.jbpatch.PatchMetadata.PatchableClass;
import java.util.Map;
import serp.bytecode.BCClass;
import serp.bytecode.BCField;

public class MoreOptionsPatch extends Patch {
  private static final String CLASS = "com.amazon.ebook.booklet.reader.resources.ReaderResources";
  // pw5 5.17.1.0.3
  private static final String MD5_BEFORE = "6d1f445a668c4d8128d6af7faeb0d007";
  private static final String MD5_AFTER = "d2a85c3a36d54b190714761a39912e6b";

  public int getVersion() {
    return 20251001;
  }

  public boolean isAvailable() {
    int jb = Environment.getJBPatchVersionDate();
    return jb >= 20251001;
  }

  protected void initLocalization(String locale, Map<String, String> map) {
    if (RESOURCE_ID_ENGLISH.equals(locale)) {
      map.put(I18N_JBPATCH_NAME, "Modify \"More Options\" label");
      map.put(
          I18N_JBPATCH_DESCRIPTION,
          "This is a useless, proof-of-concept patch which modifies the behavior of the More"
              + " Options highlight menu in the Kindle reader.");
    }
  }

  public PatchMetadata getMetadata() {
    PatchableClass pc = new PatchableClass(CLASS).withChecksums(MD5_BEFORE, MD5_AFTER);
    return new PatchMetadata(this).withClass(pc);
  }

  public String perform(String md5, BCClass clazz) throws Throwable {
    if (md5.equals(MD5_BEFORE)) {
      System.out.println("Patching " + clazz.getName());
      for (BCField field : clazz.getDeclaredFields()) {
        System.out.println("\t" + field.getName() + " : " + field.getType().getName());
      }
      if (clazz.isInstanceOf(ReaderResources.class)) {
        System.out.println("Yep, it's ReaderResources");
      } else {
        System.out.println("Nope, it's not ReaderResources");
      }
      return "temporary error for now";
    } else {
      return "unsupported MD5: " + md5;
    }
  }
}
