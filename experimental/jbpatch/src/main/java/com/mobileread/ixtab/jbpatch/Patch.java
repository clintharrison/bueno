package com.mobileread.ixtab.jbpatch;

import com.mobileread.ixtab.jbpatch.conf.ConfigurableSettings;
import com.mobileread.ixtab.jbpatch.resources.JBPatchResource;
import com.mobileread.ixtab.jbpatch.resources.KeyValueResource;
import com.mobileread.ixtab.jbpatch.resources.ResourceMapProvider;
import java.io.ByteArrayInputStream;
import java.io.PrintStream;
import java.security.Permission;
import java.util.Map;
import java.util.TreeMap;
import serp.bytecode.BCClass;
import serp.bytecode.Code;
import serp.bytecode.Instruction;
import serp.bytecode.Project;

/**
 * A <tt>Patch</tt> is a class that can modify one or more versions of one or more class
 * definitions, before the class definition is actually loaded by the JVM.
 *
 * <p><b>IMPORTANT</b>: If a patch includes ANY inner classes (named or anonymous), you <b>MUST</b>
 * create a jar file from it. That is because the system is expecting each patch to be a single file
 * (either a .class, or a .jar). If you create a jar, you must include a valid manifest, specifying
 * the "main class" (the actual patch).
 *
 * @author ixtab
 */
public abstract class Patch implements Comparable<Patch>, ResourceMapProvider {

  /** Patch name. You <b>MUST</b> provide a localization entry for this key. */
  public static final String I18N_JBPATCH_NAME = "jbpatch.name";

  /** Short patch description. You <b>MUST</b> provide a localization entry for this key. */
  public static final String I18N_JBPATCH_DESCRIPTION = "jbpatch.description";

  /**
   * Initializes the localization for this patch. You should populate the map with the default
   * translations you require. Note: You <b>MUST</b> at least set the I18N_* keys defined above, for
   * the english locale ("en").
   *
   * <p>Everything else is up to you, you can define as many or as few key/value pairs as you like,
   * and you can provide defaults for as many or as few languages as you want.
   *
   * <p>Note that any and all of the values you set here may be overridden by the user. This is
   * intended, as it allows for easy localization for additional languages.
   *
   * @param locale the locale String for the currently considered language. It follows the standard
   *     {@link java.util.Locale} conventions.
   * @param map the map that should be populated. Of course, every locale will have its own map, so
   *     there are no collisions between languages.
   */
  protected abstract void initLocalization(String locale, Map map);

  /**
   * Returns the version of this patch. This MUST be an integer representing the date on which the
   * patch was released: for instance, the value 20120902 stands for September 2, 2012.
   *
   * <p>This method of versioning has the advantage of being unambiguous. The disadvantage is that
   * (in theory) it is impossible to release two versions on the same calendar day, and that a
   * distinction between minor and major updates may be more difficult. Still, it is the most
   * sensible versioning scheme that I have encountered so far, and so you're stuck with it. :p
   *
   * @return the patch version, according to the standard format.
   */
  public abstract int getVersion();

  /**
   * Returns the metadata about this patch, such as which classes it supports on which devices. It
   * is vitally important that the returned information is correct, otherwise the patch may simply
   * not be applied.
   *
   * @return patch metadata.
   */
  public abstract PatchMetadata getMetadata();

  /**
   * Returns all additional permissions that the patch requires in order to function properly.
   * Override this method if your patch needs additional permissions. Be as specific as you can: if
   * you only need to read/write to a single file, then only request a tailored FilePermission, etc.
   *
   * @return additional permissions that the patch requires
   */
  public Permission[] getRequiredPermissions() {
    return null;
  }

  /**
   * If this patch is configurable, return a ConfigurableSettings object here, containing all
   * configurable settings.
   *
   * @return <tt>null</tt>, if this patch does not contain configurable settings, or a
   *     ConfigurableSettings object otherwise.
   */
  protected ConfigurableSettings initConfigurableSettings() {
    return null;
  }

  /**
   * Returns whether this patch is available in the current constellation. Patches would usually use
   * the {@link Environment#getFirmware()} or {@link Environment#getJBPatchVersionDate()} methods to
   * determine whether they can run on a particular constellation.
   *
   * @return <tt>false</tt> (default implementation) if the patch can't work in the current
   *     constellation, or <tt>true</tt> if it can. The default was deliberately chosen to be
   *     restrictive - in other words, a patch is NOT considered to work in a specific
   *     constellation, unless it guarantees that it does. If this method returns false, then the
   *     patch will be completely discarded (not considered, and not shown in the UI).
   */
  public boolean isAvailable() {
    return false;
  }

  /**
   * Actually applies the patch to the given input. The class definition can be altered in whichever
   * way you see fit.
   *
   * @param md5 the MD5 sum of the original binary class content. This is guaranteed to be among the
   *     supported MD5 values, as reported by the patch's metadata.
   * @param clazz the class corresponding to the given MD5 sum.
   * @return <b><tt>null</tt></b> if the input class has been <b>successfully</b> modified, and the
   *     patch should be applied. Any other string (error message) if the patch should not be
   *     applied, for instance because something failed unexpectedly. In the latter case, the
   *     message will be logged, and any changes to the class definition are discarded, i.e. the
   *     result is the same as if this method had never been invoked.
   */
  public abstract String perform(String md5, BCClass clazz) throws Throwable;

  /*
   * *********************************************************************
   * Below are some methods and fields which you cannot override, but which
   * you may find useful.
   * *********************************************************************
   */

  /**
   * Returns the currently configured value for the given key.
   *
   * @param key the key to retrieve the value for
   * @return the value associated with the key
   */
  protected final String getConfigured(String key) {
    return configurableSettings.getValue(key);
  }

  /**
   * Returns the localized string for the given key
   *
   * @param key the key to retrieve a localized string for
   * @return the localized value for the given key, or the key itself if no localization was found
   */
  public final String localize(String key) {
    if (localizationResource != null) {
      return localizationResource.getValue(key);
    } else {
      return key;
    }
  }

  /** The jbpatch logger (logs go to /tmp/jbpatch.log on the Kindle) */
  protected static final PrintStream logger = Log.INSTANCE;

  /**
   * Convenience method to log a single string
   *
   * @param message the string to log
   */
  protected static final void log(String message) {
    logger.println(message);
  }

  /**
   * Convenience method to dump a Serp Code object to the log. Useful while developing; do not use
   * in production code, as you will clutter the logs.
   *
   * @param code the code fragment to dump
   */
  public static void dump(Code code) {
    if (code != null) {
      Instruction[] inss = code.getInstructions();
      for (int i = 0; i < inss.length; ++i) {
        Instruction ins = inss[i];
        log(i + " " + ins);
      }
    }
  }

  /**
   * Determines the current MD5 sum of a BCClass object. Useful for determining the "after patch"
   * values for the metadata.
   *
   * @param clazz the class to calculate the MD5 sum for
   * @return the MD5 sum for the given class.
   */
  protected final String md5(BCClass clazz) {
    return MD5.getMd5String(clazz.toByteArray());
  }

  /**
   * Returns the id of the patch. This is either the fully qualified class name (for standalone
   * patches), or the fully qualified package name (for jarred patches).
   *
   * @return id the patch id.
   */
  public final String id() {
    return id;
  }

  /*
   * *********************************************************************
   * Everything below here is implementation details; you don't need to be
   * concerned with it.
   * *********************************************************************
   */

  private KeyValueResource localizationResource = null;
  private ConfigurableSettings configurableSettings = null;

  private void initResources() {
    localizationResource = JBPatchResource.getResource(this, JBPatchResource.TYPE_LOCALIZATION);
    JBPatchResource.getResource(this, JBPatchResource.TYPE_CONFIGURATION);
  }

  public final Map getDefaultResourceMap(String id) {
    if (id == RESOURCE_ID_CONFIGURATION) {
      configurableSettings = initConfigurableSettings();
      return configurableSettings;
    } else {
      TreeMap map = new TreeMap();
      initLocalization(id, map);
      if (!map.isEmpty()) {
        return map;
      }
    }
    return null;
  }

  public final ConfigurableSettings getConfigurableSettings() {
    return configurableSettings;
  }

  public final byte[] patch(String className, byte[] input, final String md5) {
    try {
      BCClass clazz = loadBCClass(input);
      // log("D: about to invoke "+id+" for "+className);
      String error = perform(md5, clazz);
      if (error == null) {
        Patches.reportActive();
        return clazz.toByteArray();
      }
      log("E: " + id + " failed to patch " + className + " (" + md5 + "): " + error);
    } catch (Throwable t) {
      log("E: " + id + " failed to patch " + className + " (" + md5 + "):");
      t.printStackTrace(logger);
    }
    return input;
  }

  static BCClass loadBCClass(byte[] input) {
    return new Project().loadClass(new ByteArrayInputStream(input));
  }

  private String id;

  final Patch setId(String id) {
    this.id = id;

    initResources();

    // for chaining
    return this;
  }

  public final String getName() {
    return localizationResource.getValue(I18N_JBPATCH_NAME);
  }

  public final String getDescription() {
    return localizationResource.getValue(I18N_JBPATCH_DESCRIPTION);
  }

  private PatchMetadata metadata = null;

  public final PatchMetadata metadata() {
    if (metadata == null) {
      metadata = getMetadata();
    }
    return metadata;
  }

  public int compareTo(Patch other) {
    return id().compareTo(other.id());
  }

  public String toString() {
    return id();
  }
}
