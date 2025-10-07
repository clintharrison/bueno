package com.mobileread.ixtab.jbpatch;

import java.io.File;
import java.lang.reflect.Method;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.jar.Attributes;
import java.util.jar.JarFile;

public final class PatchContainer {

  private PatchContainer() {}

  private static Method injectMethod;

  public static Patch instantiatePatch(File zip, ClassLoader cl, String potentialId)
      throws Throwable {
    JarFile jar = verifyAsJar(zip);
    String patchClassName = getMainClassInManifest(jar);
    if (patchClassName == null) {
      throw new IllegalStateException(potentialId + " does not specify a patch to execute");
    }
    URL zipUrl = zip.toURI().toURL();
    ensurePatchIsValid(cl, patchClassName, zipUrl);
    if (!validateName(potentialId, patchClassName)) {
      return null;
    }
    injectUrlIntoClassLoader(zipUrl, cl);
    return (Patch) cl.loadClass(patchClassName).newInstance();
  }

  private static JarFile verifyAsJar(File zip) {
    try {
      return new JarFile(zip);
    } catch (Throwable t) {
      throw new IllegalStateException("not a valid jar file: " + zip);
    }
  }

  private static String getMainClassInManifest(JarFile jar) {
    try {
      Attributes attrs = jar.getManifest().getMainAttributes();
      return attrs.getValue(Attributes.Name.MAIN_CLASS);
    } catch (Exception e) {
      throw new IllegalStateException("Unable to determine patch in " + jar);
    }
  }

  private static void ensurePatchIsValid(ClassLoader cl, String patchClassName, URL zipUrl) {
    try {
      ClassLoader childLoader = new URLClassLoader(new URL[] {zipUrl}, cl);
      Class testClass = childLoader.loadClass(patchClassName);
      if (testClass == null || !Patch.class.isAssignableFrom(testClass)) {
        throw new IllegalArgumentException();
      }
    } catch (Throwable t) {
      throw new IllegalArgumentException(
          patchClassName + " in " + zipUrl + " is not a valid Patch");
    }
  }

  private static boolean validateName(String potentialId, String patchClassName) {
    int lastDotIndex = patchClassName.lastIndexOf('.');
    if (lastDotIndex == -1) {
      throw new IllegalArgumentException(
          "Main class \""
              + patchClassName
              + "\" in patch \""
              + potentialId
              + " \" must not be in default package");
    }
    String pkgName = patchClassName.substring(0, lastDotIndex);
    if (!(pkgName + PatchRepository.EXTENSION_PATCH_JARRED).equals(potentialId)) {
      Log.INSTANCE.println(
          "W: "
              + potentialId
              + " deployed with mismatching file name, ignoring: Expected "
              + (pkgName + PatchRepository.EXTENSION_PATCH_JARRED)
              + ", but filename is "
              + potentialId);
      return false;
    }
    return true;
  }

  private static void injectUrlIntoClassLoader(URL jar, ClassLoader cl) throws Throwable {
    ensureInjectMethodIsSet(cl);
    Boolean ok = (Boolean) injectMethod.invoke(cl, new Object[] {jar});
    if (!ok.booleanValue()) {
      throw new RuntimeException("Unable to inject " + jar + " into classpath");
    }
  }

  private static void ensureInjectMethodIsSet(ClassLoader cl) throws NoSuchMethodException {
    if (injectMethod == null) {
      Class clc = cl.getClass();
      injectMethod = clc.getDeclaredMethod("injectUrl", new Class[] {URL.class});
      injectMethod.setAccessible(true);
    }
  }
}
