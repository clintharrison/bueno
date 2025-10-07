package com.mobileread.ixtab.jbpatch.resources;

public interface KeyValueResource {
  public String getValue(String key);

  public boolean setValue(String key, String value);
}
