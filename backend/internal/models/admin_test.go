package models

import "testing"

func TestAdminRolePermissions(t *testing.T) {
	cases := []struct {
		name    string
		user    *User
		allowed map[string]bool
	}{
		{"super admin", &User{StaffRole: "super_admin"}, map[string]bool{"overview": true, "users": true, "marketplace": true, "orders": true, "finance": true, "support": true, "reports": true, "audit": true, "staff": true, "settings": true}},
		{"customer care", &User{StaffRole: "customer_care", IsAdmin: true}, map[string]bool{"overview": true, "users": true, "orders": true, "support": true}},
		{"operations manager", &User{StaffRole: "operations_manager"}, map[string]bool{"overview": true, "users": true, "marketplace": true, "orders": true, "finance": true, "support": true, "reports": true}},
		{"legacy administrator", &User{IsAdmin: true}, map[string]bool{"overview": true, "users": true, "marketplace": true, "orders": true, "finance": true, "support": true, "reports": true, "audit": true, "staff": true, "settings": true}},
		{"ordinary farmer", &User{Role: "farmer"}, nil},
		{"disabled administrator", &User{StaffRole: "super_admin", IsAdmin: true, StaffDisabled: true}, nil},
		{"suspended administrator", &User{StaffRole: "super_admin", AccountStatus: "suspended"}, nil},
		{"unknown role", &User{StaffRole: "owner", IsAdmin: true}, nil},
		{"missing user", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, section := range AdminSections {
				if got := tc.user.CanAdmin(section.Key); got != tc.allowed[section.Key] {
					t.Errorf("%s access = %v", section.Key, got)
				}
			}
			if tc.user.CanAdmin("unknown") {
				t.Fatal("unknown section authorized")
			}
		})
	}
}
