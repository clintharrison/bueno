package com.mobileread.ixtab.jbpatch.composition;

import com.mobileread.ixtab.jbpatch.Log;
import com.mobileread.ixtab.jbpatch.Patch;
import com.mobileread.ixtab.jbpatch.PatchMetadata.ClassChecksum;
import com.mobileread.ixtab.jbpatch.PatchMetadata.PatchableClass;
import java.util.ArrayList;
import java.util.Collections;
import java.util.Iterator;
import java.util.LinkedList;
import java.util.List;

public class PathFinder {
  private final List states = new ArrayList();
  private final List transitions = new ArrayList();
  private final State start;
  private final LinkedList stateQueue = new LinkedList();

  public PathFinder(String unpatched) {
    start = new State(unpatched, new LinkedList());
    states.add(start);
    stateQueue.add(start);
  }

  public Patch[] findPath(Patch[] patches, String className) {

    for (int i = 0; i < patches.length; ++i) {
      Patch patch = patches[i];
      PatchableClass patchable =
          findPatchableClass(className, patch.getMetadata().supportedClasses);
      createTransitions(patch, patchable.checksums);
    }

    // we now have all the edges of the graph, but only a single startNode.
    while (!stateQueue.isEmpty()) {
      State state = (State) stateQueue.removeFirst();
      addTransitionsFrom(state);
    }
    // sort the states by the number of transitions taken (patches applied), higher is better.
    // note that this is not necessarily deterministic with respect to which path is chosen,
    // but as long as all patches act according to the rules, any of the longest paths will do.
    Collections.sort(states);
    State best = (State) states.get(0);
    if (best.appliedTransitions.size() < patches.length) {
      Log.INSTANCE.println(
          "W: There are "
              + patches.length
              + " patches registered for "
              + className
              + ", but there is no possible order in which all of them can be applied!");
      Log.INSTANCE.println(
          "W: This is likely a problem of separately developed patches. To use all patches at the"
              + " same time,");
      Log.INSTANCE.println(
          "W: make sure that there is at least one possible order to feed the output of one patch"
              + " to the next.");
    }
    Patch[] result = new Patch[best.appliedTransitions.size()];
    for (int i = 0; i < result.length; ++i) {
      result[i] = ((Transition) best.appliedTransitions.get(i)).patch;
    }
    return result;
  }

  private PatchableClass findPatchableClass(String className, List supportedClasses) {
    Iterator it = supportedClasses.iterator();
    while (it.hasNext()) {
      PatchableClass patchable = (PatchableClass) it.next();
      if (patchable.className.equals(className)) {
        return patchable;
      }
    }
    return null;
  }

  private void createTransitions(Patch patch, List checksums) {
    Iterator it = checksums.iterator();
    while (it.hasNext()) {
      ClassChecksum cs = (ClassChecksum) it.next();
      transitions.add(new Transition(patch, cs.beforePatch, cs.afterPatch));
    }
  }

  private void addTransitionsFrom(State state) {
    Iterator it = transitions.iterator();
    while (it.hasNext()) {
      Transition transition = (Transition) it.next();
      if (!transition.fromMd5.equals(state.md5)) {
        continue;
      }
      if (state.isPatchApplied(transition.patch)) {
        continue;
      }
      State nextState = State.create(state, transition);
      boolean isNew = true;
      Iterator sit = states.iterator();
      while (sit.hasNext()) {
        State cmp = (State) sit.next();
        if (cmp.isEquivalent(nextState)) {
          nextState = cmp;
          isNew = false;
          break;
        }
      }
      if (isNew) {
        states.add(nextState);
        stateQueue.add(nextState);
      }
      state.nextTransitions.add(transition);
      transition.nextStates.add(nextState);
    }
  }
}
