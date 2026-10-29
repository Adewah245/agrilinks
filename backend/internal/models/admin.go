package models

// CanAdmin is the shared server-side permission policy. Unknown sections fail closed.
func (u *User) CanAdmin(section string) bool {
	if u == nil || u.StaffDisabled || u.AccountStatus == "suspended" {
		return false
	}
	role := u.AdminRole()
	switch section {
	case "overview", "users", "orders", "support":
		return role == "super_admin" || role == "customer_care" || role == "operations_manager"
	case "marketplace", "finance", "reports":
		return role == "super_admin" || role == "operations_manager"
	case "audit", "staff", "settings":
		return role == "super_admin"
	}
	return false
}

func (u *User) AdminRole() string {
	if u == nil {
		return ""
	}
	if u.StaffRole != "" {
		return u.StaffRole
	}
	// Keep existing database/configuration administrators working after upgrade.
	if u.IsAdmin {
		return "super_admin"
	}
	return ""
}

func (u *User) CanManageUsers() bool { return u.CanAdmin("marketplace") }
