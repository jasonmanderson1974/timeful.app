package models

import "testing"

func TestNormalizeRole(t *testing.T) {
	cases := []struct {
		in   Role
		want Role
	}{
		{RoleSuperAdmin, RoleSuperAdmin},
		{RoleAdmin, RoleAdmin},
		{RoleMember, RoleMember},
		{RoleGuest, RoleGuest},
		{"", RoleMember},        // empty ⇒ member (default)
		{"nonsense", RoleMember}, // unknown ⇒ member
		{"ADMIN", RoleMember},    // case-sensitive; not a known value ⇒ member
	}
	for _, c := range cases {
		if got := NormalizeRole(c.in); got != c.want {
			t.Errorf("NormalizeRole(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRoleCapabilities(t *testing.T) {
	cases := []struct {
		role         Role
		manageUsers  bool
		invite       bool
		createEvents bool
		isSuperAdmin bool
	}{
		{RoleSuperAdmin, true, true, true, true},
		{RoleAdmin, true, true, true, false},
		{RoleMember, false, true, true, false},
		{RoleGuest, false, false, false, false},
		{"", false, true, true, false},      // empty ⇒ member
		{"bogus", false, true, true, false}, // unknown ⇒ member
	}
	for _, c := range cases {
		if got := c.role.CanManageUsers(); got != c.manageUsers {
			t.Errorf("Role(%q).CanManageUsers() = %v, want %v", c.role, got, c.manageUsers)
		}
		if got := c.role.CanInvite(); got != c.invite {
			t.Errorf("Role(%q).CanInvite() = %v, want %v", c.role, got, c.invite)
		}
		if got := c.role.CanCreateEvents(); got != c.createEvents {
			t.Errorf("Role(%q).CanCreateEvents() = %v, want %v", c.role, got, c.createEvents)
		}
		if got := c.role.IsSuperAdmin(); got != c.isSuperAdmin {
			t.Errorf("Role(%q).IsSuperAdmin() = %v, want %v", c.role, got, c.isSuperAdmin)
		}
	}
}

func TestRoleAtLeast(t *testing.T) {
	// Ascending privilege order.
	ordered := []Role{RoleGuest, RoleMember, RoleAdmin, RoleSuperAdmin}
	for i, a := range ordered {
		for j, b := range ordered {
			want := i >= j
			if got := a.AtLeast(b); got != want {
				t.Errorf("Role(%q).AtLeast(%q) = %v, want %v", a, b, got, want)
			}
		}
	}

	// Empty role ranks as member.
	if !RoleMember.AtLeast("") || !Role("").AtLeast(RoleMember) {
		t.Errorf("empty role should rank equal to member")
	}
	if Role("").AtLeast(RoleAdmin) {
		t.Errorf("empty role (member) should not be at least admin")
	}
}

func TestUserEffectiveRole(t *testing.T) {
	cases := []struct {
		role Role
		want Role
	}{
		{"", RoleMember},
		{RoleGuest, RoleGuest},
		{RoleMember, RoleMember},
		{RoleAdmin, RoleAdmin},
		{RoleSuperAdmin, RoleSuperAdmin},
	}
	for _, c := range cases {
		u := &User{Role: c.role}
		if got := u.EffectiveRole(); got != c.want {
			t.Errorf("User{Role:%q}.EffectiveRole() = %q, want %q", c.role, got, c.want)
		}
	}
}
