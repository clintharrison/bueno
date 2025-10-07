package com.mobileread.ixtab.jbpatch;

import java.io.File;
import java.io.FileInputStream;
import java.io.FileOutputStream;
import java.io.FilenameFilter;
import java.io.IOException;
import java.util.Date;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;

public class KindleDirectories {
  public static final String USERSTORE_DIRECTORY = "/mnt/us/opt/jbpatch";
  public static final String LOCAL_DIRECTORY = "/var/local/jbpatch";

  private static final int SYNC_INTERVAL_MS = 5000;
  private static final int SYNC_WAIT_AFTER_START_MS = 1000;

  private static File sourceDir;
  private static File targetDir;

  private static SynchLock synchLock = new SynchLock();

  public static void init() {
    targetDir = new File(LOCAL_DIRECTORY);
    if (!targetDir.exists()) targetDir.mkdir();
    sourceDir = new File(USERSTORE_DIRECTORY);
    new SynchThread().start();
    try {
      // give synch thread a chance to run before we try to read files
      Thread.sleep(SYNC_WAIT_AFTER_START_MS);
    } catch (InterruptedException e) {
    }
  }

  private static void copy(File source, File target) throws IOException {

    FileInputStream from = null;
    FileOutputStream to = null;
    try {
      byte[] buffer = new byte[4096];
      from = new FileInputStream(source);
      to = new FileOutputStream(target);
      for (int read = from.read(buffer); read != -1; read = from.read(buffer)) {
        to.write(buffer, 0, read);
      }
    } finally {
      if (from != null) {
        from.close();
      }
      if (to != null) {
        to.close();
      }
    }
  }

  public static boolean reverseSync() {
    synchronized (synchLock) {
      int count = 0;
      try {
        File[] fromFiles = targetDir.listFiles();
        for (int i = 0; i < fromFiles.length; ++i) {
          File from = fromFiles[i];
          if (from.isFile() && from.canRead()) {
            File to = new File(sourceDir + File.separator + from.getName());
            copy(from, to);
            ++count;
          }
        }
      } catch (Throwable t) {
        Log.INSTANCE.println("Error during reverse sync, exception follows.");
        t.printStackTrace(Log.INSTANCE);
        return false;
      }
      if (count != 0) {
        Log.INSTANCE.println("Reverse sync complete, copied " + count + " files.");
      }
      return true;
    }
  }

  public static boolean cleanup() {
    synchronized (synchLock) {
      if (!synchLock.synchronizeAllFiles) {
        synchLock.synchronizeAllFiles = true;
        try {
          synchLock.wait(SYNC_INTERVAL_MS * 3);
        } catch (InterruptedException e) {
          return false;
        }
        // the synchronization must have been performed until now,
        // resetting the "synchronizeAllSettings" flag on the way.
        return synchLock.synchronizeAllFiles == false;
      } else {
        // indicates some previous error.
        return false;
      }
    }
  }

  private static class SynchThread extends Thread {

    private final ShutdownStatus shutdown = new ShutdownStatus();
    private final Map knownFiles = new HashMap();
    private final FilenameFilter normalFileFilter =
        new PatchRepository.FilenamesFilter(true, false);
    private final FilenameFilter fullFileFilter = new PatchRepository.FilenamesFilter(true, true);

    private boolean targetWasOkLastTime = true;

    public SynchThread() {
      super("JBPatchDirectoriesSynchronizerThread");
      this.setDaemon(true);
    }

    public void run() {
      Runtime.getRuntime().addShutdownHook(new ShutdownThread(this));
      synchronized (shutdown) {
        log("I: Directory synchronization thread started");
        while (!shutdown.isRequested) {
          try {
            synchronize();
          } catch (Throwable t) {
            log("E: " + new Date() + ": unexpected error while synchronizing directories:");
            t.printStackTrace(Log.INSTANCE);
          }
          try {
            shutdown.wait(SYNC_INTERVAL_MS);
          } catch (InterruptedException e) {
          }
        }
        log("I: Directory synchronization thread exiting");
      }
    }

    private void synchronize() {

      synchronized (synchLock) {
        File[] targetFiles = listTargetFiles();
        if (targetFiles == null) return;

        File[] sourceFiles =
            sourceDir.listFiles(synchLock.synchronizeAllFiles ? fullFileFilter : normalFileFilter);
        if (sourceFiles == null) return;

        if (synchLock.synchronizeAllFiles) {
          removeObsoleteFilesInTarget(sourceFiles, targetFiles);
        }
        for (int i = 0; i < sourceFiles.length; ++i) {
          File f = sourceFiles[i];
          if (!f.isFile() || !f.canRead()) {
            continue;
          }
          synchronize(f, synchLock.synchronizeAllFiles);
        }
        if (synchLock.synchronizeAllFiles) {
          synchLock.synchronizeAllFiles = false;
          synchLock.notifyAll();
        }
      }
    }

    private File[] listTargetFiles() {
      if (!targetDir.exists()) {
        knownFiles.clear();
        if (targetWasOkLastTime) {
          log(
              "W: "
                  + new Date()
                  + ": "
                  + LOCAL_DIRECTORY
                  + " does not exist, attempting to re-create it!");
        }
        if (targetDir.mkdir()) {
          log("I: " + new Date() + ": created " + LOCAL_DIRECTORY);
          targetWasOkLastTime = true;
        } else {
          if (targetWasOkLastTime) {
            log("E: " + new Date() + ": failed to create " + LOCAL_DIRECTORY);
          }
          targetWasOkLastTime = false;
        }
      }
      return targetWasOkLastTime ? targetDir.listFiles() : null;
    }

    private void removeObsoleteFilesInTarget(File[] sourceFiles, File[] targetFiles) {
      Set sourceNames = new HashSet();
      for (int i = 0; i < sourceFiles.length; ++i) {
        sourceNames.add(sourceFiles[i].getName());
      }
      for (int i = 0; i < targetFiles.length; ++i) {
        File f = targetFiles[i];
        if (!sourceNames.contains(f.getName())) {
          knownFiles.remove(f.getName());
          if (f.delete()) {
            log("I: " + new Date() + ": deleted obsolete file " + f);
          } else {
            log("E: " + new Date() + ": unable to delete obsolete file " + f);
          }
        }
      }
    }

    private void synchronize(File source, boolean force) {
      File target = new File(targetDir + File.separator + source.getName());
      FileModificationInfo info = (FileModificationInfo) knownFiles.get(source.getName());
      if (force) {
        copyIfNeeded(source, target, false);
      } else {
        boolean copy = false;
        if (info == null) {
          copy = target.exists() ? source.lastModified() > target.lastModified() : true;
        } else {
          if (source.lastModified() != info.sourceTimestamp) {
            // log("I: " + source + " was modified");
            copy = true;
          } else {
            copy = !target.exists();
          }
        }
        if (copy) {
          copyIfNeeded(source, target, info == null);
        }
      }
    }

    private void copyIfNeeded(File source, File target, boolean checkThoroughly) {
      if (checkThoroughly) {
        if (filesAreEqual(source, target)) {
          FileModificationInfo info =
              new FileModificationInfo(source.lastModified(), target.lastModified());
          knownFiles.put(source.getName(), info);
          return;
        }
      }
      if (target.exists()) {
        if (!target.delete()) {
          log("E: " + new Date() + ": unable to delete " + target);
        }
      }
      try {
        copy(source, target);
        FileModificationInfo info =
            new FileModificationInfo(source.lastModified(), target.lastModified());
        knownFiles.put(source.getName(), info);
        log("I: " + new Date() + ": synchronized " + source.getName());
      } catch (Throwable t) {
        log("E: " + new Date() + ": error while copying " + source + " :");
        t.printStackTrace(Log.INSTANCE);
      }
    }

    private boolean filesAreEqual(File source, File target) {
      String s = getMd5(source);
      String t = getMd5(target);
      return (s != null && t != null && s.equals(t));
    }

    private String getMd5(File f) {

      try {
        byte[] bytes = new byte[(int) f.length()];
        FileInputStream fis = new FileInputStream(f);
        int read = fis.read(bytes);
        if (read != bytes.length) {
          log(
              "E: "
                  + new Date()
                  + ": error while reading "
                  + f
                  + "; expected "
                  + bytes.length
                  + " bytes, but got "
                  + read);
          return null;
        }
        String md5 = MD5.getMd5String(bytes);
        return md5;
      } catch (Throwable t) {
        return null;
      }
    }

    private void log(String msg) {
      Log.INSTANCE.println(msg);
    }
  }

  private static class ShutdownStatus {
    private volatile boolean isRequested = false;
  }

  private static class FileModificationInfo {
    private final long sourceTimestamp;

    //		private final long targetTimestamp;

    public FileModificationInfo(long sourceTimestamp, long targetTimestamp) {
      super();
      this.sourceTimestamp = sourceTimestamp;
      //			this.targetTimestamp = targetTimestamp;
    }
  }

  private static class ShutdownThread extends Thread {

    private final SynchThread sync;

    public ShutdownThread(SynchThread sync) {
      super("JBPatchSynchronizerShutdownThread");
      this.sync = sync;
    }

    public void run() {
      synchronized (sync.shutdown) {
        sync.shutdown.isRequested = true;
        sync.shutdown.notifyAll();
      }
    }
  }

  private static class SynchLock {
    private volatile boolean synchronizeAllFiles = false;
  }
}
