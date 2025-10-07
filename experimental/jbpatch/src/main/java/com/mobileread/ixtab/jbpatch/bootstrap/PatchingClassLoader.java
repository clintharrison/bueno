package com.mobileread.ixtab.jbpatch.bootstrap;

import com.mobileread.ixtab.jbpatch.Log;
import com.mobileread.ixtab.jbpatch.MD5;
import com.mobileread.ixtab.jbpatch.Patch;
import com.mobileread.ixtab.jbpatch.PatchMetadata.ClassChecksum;
import com.mobileread.ixtab.jbpatch.Patches;
import com.mobileread.ixtab.jbpatch.composition.PathFinder;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.PrintStream;
import java.lang.reflect.Field;
import java.net.JarURLConnection;
import java.net.URL;
import java.net.URLClassLoader;
import java.net.URLConnection;
import java.security.AccessControlContext;
import java.security.AccessController;
import java.security.CodeSource;
import java.security.PrivilegedExceptionAction;
import java.security.cert.Certificate;
import java.util.Date;
import java.util.Iterator;
import java.util.Map;
import java.util.jar.Attributes;
import java.util.jar.Attributes.Name;
import java.util.jar.JarEntry;
import java.util.jar.JarFile;
import java.util.jar.Manifest;

class PatchingClassLoader extends URLClassLoader {

  private static final String PACKAGE_NOFIND = "com.mobileread.ixtab.jbpatch.bootstrap.";
  private static final String PACKAGE_NOPATCH = "com.mobileread.ixtab.jbpatch.";
  private static final String LOG_CLASS_NAME = "com.mobileread.ixtab.jbpatch.Log";
  private static final String LOG_INSTANCE_NAME = "INSTANCE";

  @SuppressWarnings("removal")
  private final AccessControlContext acc;

  private final URL[] urls;
  private PrintStream log;
  private final Map avoidPackages;

  static PatchingClassLoader inject() throws Exception {
    ClassLoader replacedLoader = PatchingClassLoader.class.getClassLoader();
    if (!(replacedLoader instanceof URLClassLoader)) {
      throw new IllegalStateException();
    }

    ClassLoader parentLoader = replacedLoader.getParent();
    if (parentLoader instanceof PatchingClassLoader) {
      return (PatchingClassLoader) parentLoader;
    }

    URL[] urls = ((URLClassLoader) replacedLoader).getURLs();

    PatchingClassLoader patchLoader = new PatchingClassLoader(urls, parentLoader, replacedLoader);
    replaceParent(replacedLoader, patchLoader);
    return patchLoader;
  }

  private static void replaceParent(ClassLoader victim, PatchingClassLoader newParent)
      throws NoSuchFieldException, IllegalAccessException {
    Field f = ClassLoader.class.getDeclaredField("parent");
    f.setAccessible(true);
    f.set(victim, newParent);
  }

  @SuppressWarnings("removal")
  PatchingClassLoader(URL[] urls, ClassLoader parent, ClassLoader child) {
    super(urls, parent);
    this.urls = urls.clone();
    acc = AccessController.getContext();
    avoidPackages = getPackagesToAvoid(child);
    log = loadLog();
    onInit();
  }

  private Map getPackagesToAvoid(ClassLoader child) {
    try {
      Field pkg = ClassLoader.class.getDeclaredField("packages");
      pkg.setAccessible(true);
      return (Map) pkg.get(child);
    } catch (Throwable t) {
      throw new RuntimeException(t);
    }
  }

  private PrintStream loadLog() {
    try {
      Class clazz = loadClass(LOG_CLASS_NAME, true);
      Field instance = clazz.getDeclaredField(LOG_INSTANCE_NAME);
      return (PrintStream) instance.get(null);
    } catch (Throwable e) {
      e.printStackTrace();
    }
    return System.err;
  }

  private void onInit() {
    log("");
    log("Log start timestamp: " + new Date());
    log("Bootstrap OK, PatchingClassLoader instantiated");
    log("   Packages still handled by original ClassLoader:");
    Iterator it = avoidPackages.keySet().iterator();
    while (it.hasNext()) {
      log("   - " + it.next().toString());
    }
    log("");
  }

  @SuppressWarnings({"removal", "unchecked"})
  protected Class findClass(final String name) throws ClassNotFoundException {
    if (name.startsWith(PACKAGE_NOFIND)) throw new ClassNotFoundException();
    try {
      try {
        return (Class)
            AccessController.doPrivileged(
                new PrivilegedExceptionAction<Class>() {
                  public Class run() throws ClassNotFoundException {
                    String path = name.replace('.', '/').concat(".class");
                    try {
                      URL resUrl = findResource(path);
                      if (resUrl != null) {
                        try {
                          URLConnection conn = resUrl.openConnection();
                          Manifest man = null;
                          Certificate[] certs = null;
                          URL codeSourceURL = null;

                          if (conn instanceof JarURLConnection) {
                            JarURLConnection jconn = (JarURLConnection) conn;
                            jconn.setUseCaches(true);
                            JarFile jar = jconn.getJarFile();
                            JarEntry entry = jconn.getJarEntry();
                            man = jar.getManifest();
                            if (entry != null) {
                              certs = entry.getCertificates();
                            }
                            codeSourceURL = jconn.getJarFileURL();
                          } else {
                            // non-jar resource (directory or file). Try to determine the code
                            // source URL
                            // by matching the resource URL against this loader's URLs
                            String r = resUrl.toString();
                            for (URL u : urls) {
                              if (r.startsWith(u.toString())) {
                                codeSourceURL = u;
                                break;
                              }
                            }
                            if (codeSourceURL == null) {
                              codeSourceURL = resUrl;
                            }
                          }

                          // read bytes
                          try (InputStream is = conn.getInputStream()) {
                            byte[] b = readAllBytes(is);
                            String pkgname = getPackageName(name);
                            if (pkgname != null) {
                              if (avoidPackages.containsKey(pkgname)) {
                                throw new AvoidThisPackageException();
                              }
                              defineOrVerifyPackage(man, codeSourceURL, pkgname);
                            }
                            CodeSource cs = new CodeSource(codeSourceURL, certs);
                            if (okToPatch(name)) {
                              b = patch(name, b);
                            }
                            return defineClass(name, b, 0, b.length, cs);
                          }
                        } catch (IOException e) {
                          throw new ClassNotFoundException(name, e);
                        }
                      } else {
                        throw new ClassNotFoundException(name);
                      }
                    } catch (AvoidThisPackageException e) {
                      // wrap as CNF so outer handler can re-throw
                      ClassNotFoundException cnf = new ClassNotFoundException(name);
                      cnf.initCause(e);
                      throw cnf;
                    }
                  }
                },
                acc);
      } catch (java.security.PrivilegedActionException pae) {
        throw (ClassNotFoundException) pae.getException();
      }
    } catch (ClassNotFoundException nf) {
      Throwable cause = nf.getCause();
      if (cause != null && cause instanceof AvoidThisPackageException) {
        //				log("avoiding: "+name);
        throw nf;
      }
      // I have absolutely no idea why this actually works.
      return super.findClass(name);
    }
  }

  // no-op: resource-based path removed. Class bytes are read directly from URLConnections in
  // findClass

  private String getPackageName(String classname) {
    int i = classname.lastIndexOf('.');
    if (i == -1) {
      return null;
    }
    String pkgname = classname.substring(0, i);
    return pkgname;
  }

  private void defineOrVerifyPackage(Manifest man, URL url, String pkgname) throws IOException {
    Package pkg = getPackage(pkgname);
    if (pkg != null) {
      verifyPackageSecurity(url, pkgname, pkg, man);
    } else {
      definePackage(url, pkgname, man);
    }
  }

  private void verifyPackageSecurity(URL url, String pkgname, Package pkg, Manifest man) {
    // Package found, so check package sealing.
    if (pkg.isSealed()) {
      // Verify that code source URL is the same.
      if (!pkg.isSealed(url)) {
        throw new SecurityException("sealing violation: package " + pkgname + " is sealed");
      }

    } else {
      // Make sure we are not attempting to seal the package
      // at this code source URL.
      if ((man != null) && isSealed(pkgname, man)) {
        throw new SecurityException(
            "sealing violation: can't seal package " + pkgname + ": already loaded");
      }
    }
  }

  private void definePackage(URL url, String pkgname, Manifest man) {
    if (man != null) {
      definePackage(pkgname, man, url);
    } else {
      definePackage(pkgname, null, null, null, null, null, null, null);
    }
  }

  // legacy Resource-based defineClass removed; bytes are defined directly in findClass

  private static byte[] readAllBytes(InputStream in) throws IOException {
    ByteArrayOutputStream out = new ByteArrayOutputStream();
    byte[] buf = new byte[8192];
    int r;
    while ((r = in.read(buf)) != -1) {
      out.write(buf, 0, r);
    }
    return out.toByteArray();
  }

  private boolean okToPatch(String name) {
    return !name.startsWith(PACKAGE_NOPATCH);
  }

  Class defineClass(String name, byte[] b) {
    return defineClass(name, b, 0, b.length, (CodeSource) null);
  }

  private boolean isSealed(String name, Manifest man) {
    String path = name.replace('.', '/').concat("/");
    Attributes attr = man.getAttributes(path);
    String sealed = null;
    if (attr != null) {
      sealed = attr.getValue(Name.SEALED);
    }
    if (sealed == null) {
      if ((attr = man.getMainAttributes()) != null) {
        sealed = attr.getValue(Name.SEALED);
      }
    }
    return "true".equalsIgnoreCase(sealed);
  }

  void log(String msg) {
    log.println(msg);
    log.flush();
  }

  private byte[] patch(String name, byte[] input) {
    Patch[] patches = Patches.get(name);
    if (patches == null || patches.length == 0) {
      return input;
    }
    byte[] output = input;
    String md5 = MD5.getMd5String(input);

    if (patches.length > 1) {
      PathFinder path = new PathFinder(md5);
      patches = path.findPath(patches, name);
    }

    for (int i = 0; i < patches.length; ++i) {
      Patch p = (Patch) patches[i];
      ClassChecksum checksums = p.getMetadata().getChecksumsFor(name, md5);
      if (checksums == null) {
        Log.INSTANCE.println("E: " + p + " does not support MD5 " + md5 + " for class " + name);
        continue;
      }
      output = p.patch(name, input, md5);
      if (output != input) {
        input = output;
        String pmd5 = md5;
        md5 = MD5.getMd5String(input);
        log("I: " + p.id() + " applied to " + name + " (" + pmd5 + " -> " + md5 + ")");
        if (!md5.equals(checksums.afterPatch)) {
          log(
              "W: "
                  + p
                  + " produced MD5 \""
                  + md5
                  + "\", but declared \""
                  + checksums.afterPatch
                  + "\"");
        }
      }
    }
    return output;
  }

  boolean injectUrl(URL jar) {
    URL[] before = getURLs();
    addURL(jar);
    URL[] after = getURLs();
    return before.length + 1 == after.length;
  }

  private static class AvoidThisPackageException extends RuntimeException {
    private static final long serialVersionUID = 1L;
  }
}
