package com.mobileread.ixtab.jbpatch.resources;

import com.mobileread.ixtab.jbpatch.conf.ConfigurableSettings;
import java.util.Map;

public interface ResourceMapProvider {
  public String RESOURCE_ID_CONFIGURATION = null;
  public String RESOURCE_ID_ENGLISH = "en";

  /**
   * Returns the default resources for the given type. The result should be a TreeMap, mapping
   * Strings to Strings. If the resource type is not supported, <tt>null</tt> must be returned. For
   * patches, it is highly recommended (consider it mandatory) to provide complete implementations
   * at least for the <tt>RESOURCE_ID_ENGLISH</tt> parameter.
   *
   * @param id the type of resource that is requested. This may either by <tt>null</tt>
   *     (<tt>RESOURCE_ID_CONFIGURATION</tt>), if the default configuration is requested, or a
   *     locale String such as "en" or "de_CH".
   * @return a TreeMap containing String-to-String mappings, or <tt>null</tt> if there are no
   *     predefined defaults for the requested resource type.
   */
  public Map getDefaultResourceMap(String id);

  /**
   * Returns the globally unique ID of this resource provider. It is <i>highly</i> recommended to
   * use something like the Java class (or package) name, to make sure that the risk of conflicts is
   * as low as possible, while still providing some semantics within the ID.
   *
   * @return the globally unique ID of this resource provider.
   */
  public String id();

  public ConfigurableSettings getConfigurableSettings();
}
