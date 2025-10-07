package com.mobileread.ixtab.jbpatch;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.URL;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.Map;
import java.util.TreeMap;

public class I18n {

  private final String[] locales;
  private final Map[] translations;

  public I18n() {
    this((InputStream) null, null);
  }

  public I18n(URL url) {
    this(url, null);
  }

  public I18n(URL url, Locale locale) {
    this(safeStream(url), locale);
  }

  public I18n(InputStream stream) {
    this(stream, null);
  }

  public I18n(InputStream stream, Locale locale) {
    locales = determineLocalesSearchOrder(locale != null ? locale : Locale.getDefault());
    translations = initializeTranslations(locales.length);
    if (stream != null) {
      fillTranslations(stream);
    }
  }

  private static InputStream safeStream(URL url) {
    if (url == null) return null;
    try {
      return url.openStream();
    } catch (IOException e) {
      return null;
    }
  }

  private String[] determineLocalesSearchOrder(Locale current) {
    List list = new ArrayList(3);
    list.add(current);
    Locale last = current;
    current = new Locale(last.getLanguage());
    if (!current.equals(last)) {
      list.add(current);
    }
    if (!current.equals(Locale.ENGLISH)) {
      list.add(Locale.ENGLISH);
    }

    String[] result = new String[list.size()];
    for (int i = 0; i < result.length; ++i) {
      result[i] = ((Locale) list.get(i)).toString();
    }
    return result;
  }

  private Map[] initializeTranslations(int count) {
    Map[] map = new Map[count];
    for (int i = 0; i < count; ++i) {
      map[i] = new TreeMap();
    }
    return map;
  }

  private void fillTranslations(InputStream stream) {
    BufferedReader r = null;
    try {
      r = new BufferedReader(new InputStreamReader(stream, "UTF-8"));
      Map translation = null;
      for (String line = r.readLine(); line != null; line = r.readLine()) {
        if (isIgnorable(line)) continue;
        String localeCode = interpretAsSection(line);
        if (localeCode != null) {
          translation = lookupTranslationFor(localeCode);
          continue;
        }
        insertIfValid(line, translation);
      }
    } catch (IOException e) {
    } finally {
      if (r != null) {
        try {
          r.close();
        } catch (IOException e) {
        }
      }
    }
  }

  private boolean isIgnorable(String line) {
    String trimmed = line.trim();
    return trimmed.length() == 0 || trimmed.startsWith("#");
  }

  private String interpretAsSection(String line) {
    // allow trailing spaces, but not leading ones
    String trimmed = line.trim();
    if (line.startsWith("[") && trimmed.endsWith("]")) {
      return trimmed.substring(1, trimmed.length() - 1);
    }
    return null;
  }

  private Map lookupTranslationFor(String localeCode) {
    for (int i = 0; i < locales.length; ++i) {
      if (locales[i].equals(localeCode)) {
        return translations[i];
      }
    }
    return null;
  }

  private void insertIfValid(String line, Map translation) {
    if (translation == null) return;
    int delim = line.indexOf('=');
    if (delim == -1) return;
    String key = line.substring(0, delim).trim();
    String value = line.substring(delim + 1).trim();
    translation.put(key, value);
  }

  public String[] getOrderedLocales() {
    String[] copy = new String[locales.length];
    System.arraycopy(locales, 0, copy, 0, locales.length);
    return copy;
  }

  public String i18n(String key) {
    key = key.trim();

    for (int i = 0; i < translations.length; ++i) {
      Object match = translations[i].get(key);
      if (match != null) {
        return (String) match;
      }
    }
    return key;
  }
}
