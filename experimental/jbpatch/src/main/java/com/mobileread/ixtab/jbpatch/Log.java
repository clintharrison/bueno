package com.mobileread.ixtab.jbpatch;

import java.io.FileOutputStream;
import java.io.IOException;
import java.io.PrintStream;

public final class Log {

  private Log() {}

  public static final String LOGFILE = "/tmp/jbpatch.log";

  public static final PrintStream INSTANCE = instantiate();

  private static PrintStream instantiate() {
    if (Environment.isKindle()) {
      try {
        return new PrintStream(new FileOutputStream(LOGFILE));
      } catch (IOException e) {
      }
    }
    return System.err;
  }
}
