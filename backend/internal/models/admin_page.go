package models

type AdminSection struct{ Key, Title, Description, Path string }
type AdminMetric struct{ Label, Value, Detail, Path string }
type AdminRow struct {
	ID, Status string
	Cells      []string
}
type AdminTable struct {
	Key, Title string
	Columns    []string
	Rows       []AdminRow
	HasNext    bool
}
type AdminStaffOption struct {
	ID         int
	Name, Role string
}
type AdminSetting struct{ Key, Label, Value string }
type AdminPage struct {
	User                         *User
	Section                      AdminSection
	Navigation                   []AdminSection
	Metrics                      []AdminMetric
	Tables                       []AdminTable
	Staff                        []AdminStaffOption
	Settings                     []AdminSetting
	Policies                     []AdminSetting
	Search, Error, Notice, CSRF  string
	Page, PreviousPage, NextPage int
	HasNext                      bool
}

var AdminSections = []AdminSection{
	{"overview", "Dashboard Overview", "Monitor platform activity, pending tasks and alerts.", "/admin"},
	{"users", "Users & Farmer Management", "Farmers, buyers, account status, verification progress and reported accounts.", "/admin/users"},
	{"marketplace", "Marketplace Management", "Products, categories, listing reports and suspicious activity.", "/admin/marketplace"},
	{"orders", "Orders & Deliveries", "Order status, delivery tracking, cancellations and disputes.", "/admin/orders"},
	{"finance", "Finance & Payments", "Wallets, deposits, withdrawals, commissions, refunds and settlement records.", "/admin/finance"},
	{"support", "Customer Care & Disputes", "Support tickets, complaints, assigned staff and resolution history.", "/admin/support"},
	{"reports", "Reports & Analytics", "Marketplace activity, sales, transaction summaries, farmer growth and operational performance.", "/admin/reports"},
	{"audit", "Audit Logs", "Who performed an action, what changed, when it happened and whether it succeeded.", "/admin/audit"},
	{"staff", "Staff, Roles & Permissions", "Create staff accounts, assign roles, restrict access and disable compromised accounts.", "/admin/staff"},
	{"settings", "Platform Settings", "Approved system settings, notification templates and operational policies.", "/admin/settings"},
}
