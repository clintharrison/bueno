package com.mobileread.ixtab.jbpatch;

import com.mobileread.ixtab.jbpatch.PatchMetadata.PatchableClass;
import com.mobileread.ixtab.jbpatch.builtin.DeviceInfoPatch;
import java.io.BufferedInputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.InputStream;
import java.io.PrintStream;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.List;
import java.util.Map;
import java.util.Map.Entry;
import serp.bytecode.BCClass;
import serp.bytecode.Project;

public class Patches {

  private static final String PCL_CLASSNAME =
      "com.mobileread.ixtab.jbpatch.bootstrap.PatchingClassLoader";
  private static final String PCL_DEFINECLASS_METHODNAME = "defineClass";

  private static final Map patches = new HashMap();
  private static final PrintStream log = Log.INSTANCE;

  private static int active = 0;
  private static int available = 0;

  private Patches() {}

  private static void log(String msg) {
    log.println(msg);
  }

  public static Patch[] get(String className) {
    List result = (List) patches.get(className);
    if (result == null) {
      return null;
    }
    Object[] objects = result.toArray();
    Patch[] patches = new Patch[objects.length];
    for (int i = 0; i < objects.length; ++i) {
      patches[i] = (Patch) objects[i];
    }
    return patches;
  }

  private static boolean enable(Patch patch) {

    if (!patch.isAvailable()) {
      log(
          "W: "
              + patch.id()
              + " does not support current device/JBPatch constellation, patch disabled");
      return false;
    }

    List classes = patch.metadata().supportedClasses;

    if (classes != null) {
      for (int i = 0; i < classes.size(); ++i) {
        PatchableClass clazz = (PatchableClass) classes.get(i);
        registerForClass(clazz, patch);
      }
      int permissions = PatchPolicy.register(patch);
      if (permissions > 0) {
        log(
            "I: "
                + patch.id()
                + " has been granted "
                + permissions
                + " additional permission"
                + (permissions == 1 ? "" : "s")
                + " on request");
      }
      return true;
    }
    return false;
  }

  private static void registerForClass(PatchableClass pclass, Patch patch) {
    String className = pclass.className;
    List handlers = getNonNullList(className);
    handlers.add(patch);
    log("I: " + patch.id() + " registered for " + className);
    ++available;
  }

  private static List getNonNullList(String className) {
    List list = (List) patches.get(className);
    if (list == null) {
      list = new ArrayList();
      patches.put(className, list);
    }
    return list;
  }

  private static Patch add(File file, boolean enable) {
    ClassLoader cl = Patches.class.getClassLoader();
    if (isPatchingClassLoader(cl)) {
      String id = file.getName();
      try {
        Patch p = null;
        String shortId = null;
        if (id.endsWith(PatchRepository.EXTENSION_PATCH_STANDALONE)) {
          shortId =
              id.substring(0, id.length() - PatchRepository.EXTENSION_PATCH_STANDALONE.length());
          p = instantiate(new BufferedInputStream(new FileInputStream(file)), cl, id);
        } else if (id.endsWith(PatchRepository.EXTENSION_PATCH_JARRED)) {
          shortId = id.substring(0, id.length() - PatchRepository.EXTENSION_PATCH_JARRED.length());
          p = PatchContainer.instantiatePatch(file, cl, id);
        } else {
          throw new IllegalStateException();
        }
        if (p != null) {
          p.setId(shortId);
          if (!enable) {
            return p;
          }
          if (enable && enable(p)) {
            return p;
          }
        }
      } catch (Throwable t) {
        log("E: while instantiating " + id + ": ");
        t.printStackTrace(log);
      }
    }
    return null;
  }

  private static boolean isPatchingClassLoader(ClassLoader cl) {
    return cl.getClass().getName().equals(PCL_CLASSNAME);
  }

  private static Patch instantiate(InputStream stream, ClassLoader cl, String potentialId)
      throws Throwable {
    BCClass bc = new Project().loadClass(stream);
    String name = bc.getName();
    byte[] bytes = bc.toByteArray();
    try {
      return instantiate(name, bytes, cl, potentialId);
    } catch (InvocationTargetException e) {
      Throwable cause = e.getCause();
      if (cause != null && cause instanceof LinkageError) {
        try {
          Class.forName(name);
          log(
              "E: "
                  + potentialId
                  + " not instantiated: another class with name "
                  + name
                  + " is already loaded");
          return null;
        } catch (Throwable t) {
        }
      }
      throw e;
    }
  }

  private static Patch instantiate(
      String name, byte[] bytes, ClassLoader cl, String containerFileName) throws Throwable {
    Class c = defineClass(cl, name, bytes);
    if (!Patch.class.isAssignableFrom(c)) {
      log("E: " + name + " is not a valid patch.");
      return null;
    }
    if (!containerFileName.equals(c.getName() + PatchRepository.EXTENSION_PATCH_STANDALONE)) {
      log(
          "W: "
              + c.getName()
              + " deployed with mismatching file name, ignoring: Expected "
              + c.getName()
              + PatchRepository.EXTENSION_PATCH_STANDALONE
              + ", but filename is "
              + containerFileName);
      return null;
    }
    return (Patch) c.newInstance();
  }

  private static Method defineClassMethod = null;

  private static synchronized Class defineClass(ClassLoader cl, String name, byte[] bytes)
      throws Throwable {

    ensureDefineClassMethodIsSet(cl);

    return (Class) defineClassMethod.invoke(cl, new Object[] {name, bytes});
  }

  private static void ensureDefineClassMethodIsSet(ClassLoader cl) throws NoSuchMethodException {
    if (defineClassMethod == null) {
      Class clc = cl.getClass();
      defineClassMethod =
          clc.getDeclaredMethod(
              PCL_DEFINECLASS_METHODNAME, new Class[] {String.class, byte[].class});
      defineClassMethod.setAccessible(true);
    }
  }

  private static boolean initialized = false;

  public static void init() {
    if (initialized) {
      return;
    }
    log("JBPatch version " + JBPatchMetadata.VERSION + ": initializing patches");
    initialized = true;
    log("");
    KindleDirectories.init();
    addBuiltins();
    addExternals();
  }

  private static void addBuiltins() {
    if (!enable(((Patch) new DeviceInfoPatch()).setId("(builtin) DeviceInfo"))) {
      log("E: built-in device info patch could not be enabled!");
    }
  }

  private static void addExternals() {
    PatchRepository repo = PatchRepository.getInstance();
    Map patchMap = repo.initialize();
    Iterator it = patchMap.entrySet().iterator();
    while (it.hasNext()) {
      Map.Entry entry = (Entry) it.next();
      File file = (File) entry.getKey();
      Boolean active = (Boolean) entry.getValue();
      Patch patch = add(file, active.booleanValue());
      if (patch != null) {
        repo.addAvailable(patch, active);
      }
    }
  }

  static void reportActive() {
    ++active;
  }

  public static int getActiveCount() {
    return active;
  }

  public static int getAvailableCount() {
    return available;
  }
}
