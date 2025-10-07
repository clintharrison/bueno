package com.mobileread.ixtab.jbpatch;

import java.util.ArrayList;
import java.util.Iterator;
import java.util.List;

/**
 * This class represents metadata about a particular patch, in particular, which exact versions of
 * which exact classes are supported. The information is represented in a hierarchical top-down
 * manner, i.e. this particular class only contains references to the supported classes, and each
 * {@link PatchableClass} instance contains references to the supported versions.
 *
 * @author ixtab
 */
public final class PatchMetadata {

  public final Patch patch;
  public final List supportedClasses = new ArrayList();

  public PatchMetadata(Patch patch) {
    this.patch = patch;
  }

  /**
   * Adds information about a supported {@link PatchableClass} to this metadata.
   *
   * @param clazz a {@link PatchableClass} supported by the patch owning this PatchMetadata object.
   * @return the current PatchMetadata object, so that invocations can be chained.
   */
  public PatchMetadata withClass(PatchableClass clazz) {
    supportedClasses.add(clazz);
    return this;
  }

  //	/**
  //	 * This class represents information about a particular device, and an
  //	 * arbitrary number of classes, supported by the {@link Patch} that produced
  //	 * the encompassing {@link PatchMetadata}.
  //	 */
  //	public static final class PatchableDevice {
  //		public final KindleDevice device;
  //		public final List supportedClasses = new ArrayList();
  //
  //		/**
  //		 * Instantiates a new PatchableDevice, referring to the given
  //		 * {@link KindleDevice}.
  //		 *
  //		 * @param device
  //		 *            a particular device supported by the instantiating patch.
  //		 */
  //		public PatchableDevice(KindleDevice device) {
  //			this.device = device;
  //		}
  //
  //		/**
  //		 * Adds information about a supported {@link PatchableClass}.
  //		 *
  //		 * @param clazz
  //		 *            a {@link PatchableClass} supported by the owning patch.
  //		 * @return the current PatchableDevice object, so that invocations can
  //		 *         be chained.
  //		 */
  //		public PatchableDevice withClass(PatchableClass clazz) {
  //			supportedClasses.add(clazz);
  //			return this;
  //		}
  //
  //	}

  /**
   * This class represents information about a particular class (name), with an arbitrary number of
   * class "expressions" (possible different binary states), supported by the owning Patch. In
   * short, a patch may, even on the same device, support multiple variants of the same basic class
   * (in terms of their bytecode's MD5 sum). This might be relevant if a class's bytecode is (or may
   * be) patched multiple times before actually being loaded by the class loader.
   */
  public static final class PatchableClass {
    public final String className;
    public final List checksums = new ArrayList();

    public PatchableClass(String className) {
      this.className = className;
    }

    /**
     * Indicates the binary checksums of the concerned class's bytecode before, and after, it is
     * patched.
     *
     * @param before the MD5 sum of the class's bytecode, before it could be patched
     * @param after the MD5 sum of the class's bytecode, after it would have been patched
     * @return the containing PatchableClass instance, to allow for method chaining
     */
    public PatchableClass withChecksums(String before, String after) {
      checksums.add(new ClassChecksum(before, after));
      return this;
    }

    public ClassChecksum getChecksumsFor(String md5) {
      Iterator csit = checksums.iterator();
      while (csit.hasNext()) {
        ClassChecksum cs = (ClassChecksum) csit.next();
        if (cs.beforePatch.equals(md5)) {
          return cs;
        }
      }
      return null;
    }
  }

  public static final class ClassChecksum {
    public final String beforePatch;
    public final String afterPatch;

    public ClassChecksum(String beforePatch, String afterPatch) {
      super();
      this.beforePatch = beforePatch;
      this.afterPatch = afterPatch;
    }
  }

  public ClassChecksum getChecksumsFor(String className, String md5) {
    Iterator classes = supportedClasses.iterator();
    while (classes.hasNext()) {
      PatchableClass patchable = (PatchableClass) classes.next();
      if (patchable.className.equals(className)) {
        return patchable.getChecksumsFor(md5);
      }
    }
    return null;
  }
}
