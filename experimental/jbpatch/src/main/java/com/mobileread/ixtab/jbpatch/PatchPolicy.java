package com.mobileread.ixtab.jbpatch;

import java.security.CodeSource;
import java.security.Permission;
import java.security.PermissionCollection;
import java.security.Permissions;
import java.security.Policy;
import java.security.ProtectionDomain;
import java.util.Enumeration;
import java.util.HashMap;
import java.util.Map;

@SuppressWarnings("removal")
public class PatchPolicy extends Policy {

  private static final PatchPolicy instance;

  static {
    instance = new PatchPolicy(getPolicy());
    Log.INSTANCE.println("I: Patch policy initialized");
  }

  private final Policy delegate;
  private final Map<ProtectionDomain, PermissionCollection> extendedPolicies = new HashMap<>();

  private PatchPolicy(Policy delegate) {
    this.delegate = delegate;
    Policy.setPolicy(this);
  }

  public static int register(Patch patch) {
    Permission[] required = patch.getRequiredPermissions();
    return instance.register(patch, required);
  }

  private int register(Patch patch, Permission[] additional) {
    if (additional != null && additional.length != 0) {
      ProtectionDomain domain = patch.getClass().getProtectionDomain();
      extendedPolicies.put(domain, combinePermissions(additional, delegate.getPermissions(domain)));
      return additional.length;
    }
    return 0;
  }

  private PermissionCollection combinePermissions(Permission[] p1, PermissionCollection p2) {
    Permissions combined = new Permissions();
    copyAllPermissions(p1, combined);
    copyPermissions(p2, combined);
    combined.setReadOnly();
    return combined;
  }

  private void copyAllPermissions(Permission[] source, Permissions target) {
    for (int i = 0; i < source.length; ++i) {
      target.add(source[i]);
    }
  }

  private void copyPermissions(PermissionCollection source, Permissions target) {
    Enumeration permissions = source.elements();
    while (permissions.hasMoreElements()) {
      target.add((Permission) permissions.nextElement());
    }
  }

  public PermissionCollection getPermissions(ProtectionDomain domain) {
    PermissionCollection overridden = extendedPolicies.get(domain);
    if (overridden != null) {
      return overridden;
    }
    return delegate.getPermissions(domain);
  }

  public PermissionCollection getPermissions(CodeSource codesource) {
    return delegate.getPermissions(codesource);
  }

  public boolean implies(ProtectionDomain domain, Permission permission) {
    return getPermissions(domain).implies(permission);
  }

  public void refresh() {}
}
