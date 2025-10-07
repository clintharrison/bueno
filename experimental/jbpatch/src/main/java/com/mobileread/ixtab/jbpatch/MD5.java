package com.mobileread.ixtab.jbpatch;

import java.security.MessageDigest;
import java.security.NoSuchAlgorithmException;

public class MD5 {

  private static final MessageDigest digest = getMd5Instance();

  private static MessageDigest getMd5Instance() {
    try {
      return MessageDigest.getInstance("MD5");
    } catch (NoSuchAlgorithmException e) {
      throw new RuntimeException(e);
    }
  }

  public static synchronized String getMd5String(byte[] input) {
    digest.reset();
    digest.update(input);
    byte[] b = digest.digest();
    return asString(b);
  }

  private static String asString(byte[] bytes) {
    StringBuffer r = new StringBuffer();
    for (int i = 0; i < bytes.length; ++i) {
      int b = bytes[i];
      append(r, b & 0xff);
    }
    return r.toString();
  }

  private static void append(StringBuffer r, int b) {
    String c = Integer.toHexString(b);
    if (c.length() < 2) {
      r.append("0");
    }
    r.append(c);
  }
}
