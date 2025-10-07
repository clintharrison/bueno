package com.mobileread.ixtab.jbpatch.resources;

import java.io.File;

public class ConfigurationResource implements KeyValueResource {

  private final KeyValueFile backend;

  public ConfigurationResource(ResourceMapProvider provider) {
    File file = JBPatchResource.loadOrCreateFile(provider.id(), null, provider);
    if (file != null) {
      backend = new KeyValueFile(KeyValueFile.FLAG_WRITABLE, file);
    } else {
      backend = null;
    }
  }

  public String getValue(String key) {
    return backend != null ? backend.getValue(key) : null;
  }

  public boolean setValue(String key, String value) {
    throw new UnsupportedOperationException();
  }
}
