package routes

import (
	"testing"

	"schej.it/server/models"
)

func TestIsValidEmail(t *testing.T) {
	valid := []string{
		"a@b.com",
		"stf-admin@jasonmanderson.com",
		"user@example.co.uk",
	}
	for _, e := range valid {
		if !isValidEmail(e) {
			t.Errorf("isValidEmail(%q) = false, want true", e)
		}
	}

	invalid := []string{
		"",                    // empty
		"plainaddress",        // no @
		"user+tag@gmail.com",  // '+' aliases rejected
		"<a@b.com>",           // angle-bracket form (addr.Address != input)
		"Name <a@b.com>",      // display name form
		"a b@c.com",           // space in local part
	}
	for _, e := range invalid {
		if isValidEmail(e) {
			t.Errorf("isValidEmail(%q) = true, want false", e)
		}
	}
}

func TestCanGrantRole(t *testing.T) {
	cases := []struct {
		name          string
		actor, target models.Role
		want          bool
	}{
		// Super admin actor: may grant any role EXCEPT super admin.
		{"superadmin grants guest", models.RoleSuperAdmin, models.RoleGuest, true},
		{"superadmin grants member", models.RoleSuperAdmin, models.RoleMember, true},
		{"superadmin grants admin", models.RoleSuperAdmin, models.RoleAdmin, true},
		{"superadmin grants superadmin", models.RoleSuperAdmin, models.RoleSuperAdmin, false},

		// Admin actor: may grant guest/member/admin (incl. admin), never super admin.
		{"admin grants guest", models.RoleAdmin, models.RoleGuest, true},
		{"admin grants member", models.RoleAdmin, models.RoleMember, true},
		{"admin grants admin", models.RoleAdmin, models.RoleAdmin, true},
		{"admin grants superadmin", models.RoleAdmin, models.RoleSuperAdmin, false},

		// Member actor: may grant guests only.
		{"member grants guest", models.RoleMember, models.RoleGuest, true},
		{"member grants member", models.RoleMember, models.RoleMember, false},
		{"member grants admin", models.RoleMember, models.RoleAdmin, false},
		{"member grants superadmin", models.RoleMember, models.RoleSuperAdmin, false},

		// Guest actor: may grant nothing.
		{"guest grants guest", models.RoleGuest, models.RoleGuest, false},
		{"guest grants member", models.RoleGuest, models.RoleMember, false},

		// Empty actor role normalizes to member ⇒ guests only.
		{"empty actor grants guest", "", models.RoleGuest, true},
		{"empty actor grants member", "", models.RoleMember, false},
	}
	for _, c := range cases {
		if got := canGrantRole(c.actor, c.target); got != c.want {
			t.Errorf("%s: canGrantRole(%q, %q) = %v, want %v", c.name, c.actor, c.target, got, c.want)
		}
	}
}
