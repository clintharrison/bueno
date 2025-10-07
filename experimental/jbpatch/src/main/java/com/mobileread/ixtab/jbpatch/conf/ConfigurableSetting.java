package com.mobileread.ixtab.jbpatch.conf;

public abstract class ConfigurableSetting {
  public final String name;
  public final String description;
  public final String key;
  public final String defaultValue;
  public final String hint;

  public ConfigurableSetting(
      String name, String description, String hint, String key, String defaultValue) {
    this.name = name;
    this.description = description;
    this.hint = hint;
    this.key = key;
    this.defaultValue = defaultValue;
  }

  /**
   * Returns a SettingPanel which can be used to configure this setting. You <b>must</b> create a
   * new instance at every invocation; in other words, do not try to initialize the panel at
   * instantiation: in the worst case, this could brick the device!
   *
   * <p>(Background info): If you try to initialize a JPanel at patch instantiation time, the UI
   * part is not loaded yet, which will cause the Kindle startup to crash, enter a reboot loop, and
   * eventually "need repair". You have been warned.
   *
   * @param listener the SettingChangeListener to notify on setting changes.
   * @return a {@link SettingPanel} instance.
   */
  // public abstract SettingPanel getPanel(SettingChangeListener listener);

  public String toString() {
    return "" + defaultValue;
  }

  public abstract boolean isValid(String value);

  /**
   * Returns a localized version of a given value. The default implementation returns the value
   * unchanged.
   *
   * @param value - a value to localize
   * @return the localized value
   */
  public String getLocalized(String value) {
    return value;
  }
}
