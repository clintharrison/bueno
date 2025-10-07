package com.mobileread.ixtab.jbpatch.composition;

import com.mobileread.ixtab.jbpatch.Patch;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.Iterator;
import java.util.List;

public class State implements Comparable<State> {
  public final String md5;
  public final List<Transition> appliedTransitions;
  public final List<Transition> nextTransitions = new ArrayList<>();

  public State(String md5, List<Transition> appliedTransitions) {
    this.md5 = md5;
    this.appliedTransitions = appliedTransitions;
  }

  public static State create(State previous, Transition via) {
    List<Transition> applied = new ArrayList<>(previous.appliedTransitions);
    applied.add(via);
    return new State(via.toMd5, applied);
  }

  public String toString() {
    return "State [md5="
        + md5
        + ", appliedTransitions="
        + appliedTransitions
        + ", nextTransitions="
        + nextTransitions
        + "]";
  }

  public int compareTo(State other) {
    if (appliedTransitions.size() == other.appliedTransitions.size()) {
      // no preference, both seem to do the job. Just be deterministic:
      return toString().compareTo(other.toString());
    }
    int order =
        Integer.valueOf(appliedTransitions.size())
            .compareTo(Integer.valueOf(other.appliedTransitions.size()));
    // higher is better;
    return -order;
  }

  public boolean isEquivalent(State other) {
    Patch[] myPatches = getAppliedPatches(this);
    Patch[] othersPatches = getAppliedPatches(other);
    return Arrays.equals(myPatches, othersPatches);
  }

  private static Patch[] getAppliedPatches(State state) {
    Patch[] p = new Patch[state.appliedTransitions.size()];
    for (int i = 0; i < p.length; ++i) {
      p[i] = ((Transition) state.appliedTransitions.get(i)).patch;
    }
    Arrays.sort(p);
    return p;
  }

  public boolean isPatchApplied(Patch patch) {
    Iterator it = appliedTransitions.iterator();
    while (it.hasNext()) {
      Transition t = (Transition) it.next();
      if (t.patch.equals(patch)) {
        return true;
      }
    }
    return false;
  }
}
