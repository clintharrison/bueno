package com.mobileread.ixtab.jbpatch.bootstrap;

import org.osgi.framework.BundleActivator;
import org.osgi.framework.BundleContext;

public class Stage1 implements BundleActivator {

  private boolean started = false;

  public void start(BundleContext arg0) throws Exception {
    if (!started) {
      started = true;
      PatchingClassLoader.inject();

      /*
       * For some reason, the class loading process screws up if we have a
       * direct reference to the Patches class here. So we just take one more
       * indirection.
       */
      new Stage2().run();
    }
  }

  public void stop(BundleContext arg0) throws Exception {}
}
