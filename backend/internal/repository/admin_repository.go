package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"backend/internal/models"
)

type AdminRepository struct{ db *sql.DB }

func NewAdminRepository(db *sql.DB) *AdminRepository { return &AdminRepository{db: db} }

func (r *AdminRepository) WalletSnapshot(ctx context.Context, kind string, id int64) (string, error) {
	query := `SELECT jsonb_build_object('status',status,'approved_by',approved_by,'amount',amount,'currency',currency)::text FROM wallet_transfers WHERE id=$1`
	if kind == "withdrawal" {
		query = `SELECT jsonb_build_object('status',status,'approved_by',approved_by,'amount',amount,'currency',currency)::text FROM wallet_withdrawals WHERE id=$1`
	}
	var value string
	err := r.db.QueryRowContext(ctx, query, id).Scan(&value)
	return value, err
}

type adminQuery struct {
	key, title, query string
	columns           []string
}

func (r *AdminRepository) Page(ctx context.Context, user *models.User, section models.AdminSection, search string, page int) (*models.AdminPage, error) {
	if !user.CanAdmin(section.Key) {
		return nil, ErrAdminForbidden
	}
	data := &models.AdminPage{User: user, Section: section, Search: search, Page: page, PreviousPage: page - 1, NextPage: page + 1}
	for _, item := range models.AdminSections {
		if user.CanAdmin(item.Key) {
			data.Navigation = append(data.Navigation, item)
		}
	}
	if section.Key == "overview" || section.Key == "reports" {
		metrics, err := r.metrics(ctx, user)
		if err != nil {
			return nil, err
		}
		data.Metrics = metrics
	}
	for _, spec := range adminQueries(section.Key) {
		rows, err := r.db.QueryContext(ctx, spec.query, search, (page-1)*25)
		if err != nil {
			return nil, fmt.Errorf("admin %s: %w", spec.key, err)
		}
		table := models.AdminTable{Key: spec.key, Title: spec.title, Columns: spec.columns}
		for rows.Next() {
			row := models.AdminRow{Cells: make([]string, len(spec.columns))}
			dest := []any{&row.ID, &row.Status}
			for i := range row.Cells {
				dest = append(dest, &row.Cells[i])
			}
			if err := rows.Scan(dest...); err != nil {
				rows.Close()
				return nil, err
			}
			table.Rows = append(table.Rows, row)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		if len(table.Rows) > 25 {
			table.HasNext = true
			table.Rows = table.Rows[:25]
			data.HasNext = true
		}
		data.Tables = append(data.Tables, table)
	}
	if section.Key == "support" {
		rows, err := r.db.QueryContext(ctx, `SELECT id,full_name,staff_role FROM users WHERE deleted_at IS NULL AND account_status='active' AND NOT staff_disabled AND (staff_role<>'' OR is_admin) ORDER BY full_name`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var staff models.AdminStaffOption
			if err := rows.Scan(&staff.ID, &staff.Name, &staff.Role); err != nil {
				return nil, err
			}
			data.Staff = append(data.Staff, staff)
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	if section.Key == "settings" || section.Key == "support" || section.Key == "marketplace" {
		rows, err := r.db.QueryContext(ctx, `SELECT key,value FROM admin_platform_settings ORDER BY key`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var setting models.AdminSetting
			if err := rows.Scan(&setting.Key, &setting.Value); err != nil {
				return nil, err
			}
			setting.Label = adminSettingLabels[setting.Key]
			if section.Key == "settings" {
				data.Settings = append(data.Settings, setting)
			} else if section.Key == "support" && (setting.Key == "dispute_policy" || setting.Key == "support_response_hours") {
				data.Policies = append(data.Policies, setting)
			} else if section.Key == "marketplace" && setting.Key == "listing_policy" {
				data.Policies = append(data.Policies, setting)
			}
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func (r *AdminRepository) metrics(ctx context.Context, user *models.User) ([]models.AdminMetric, error) {
	type metricQuery struct{ label, detail, path, query string }
	queries := []metricQuery{
		{"Total users", "Accounts excluding closed accounts", "/admin/users", `SELECT count(*) FROM users WHERE deleted_at IS NULL`},
		{"Active farmers", "Farmers with enabled accounts", "/admin/users?q=farmer", `SELECT count(*) FROM users WHERE role='farmer' AND deleted_at IS NULL AND account_status='active' AND NOT staff_disabled`},
		{"Buyers", "Registered buyer accounts", "/admin/users?q=buyer", `SELECT count(*) FROM users WHERE role='buyer' AND deleted_at IS NULL`},
		{"Listings", "Active marketplace listings", "/admin/marketplace", `SELECT count(*) FROM products WHERE status='active'`},
		{"Orders", "All marketplace orders", "/admin/orders", `SELECT count(*) FROM orders`},
		{"Pending tasks", "Open tickets and pending verifications", "/admin/support", `SELECT (SELECT count(*) FROM admin_support_tickets WHERE status IN ('open','in_progress'))+(SELECT count(*) FROM user_verifications WHERE verification_level='pending')`},
		{"Alerts", "Unresolved account and listing reports", "/admin/users", `SELECT count(*) FROM admin_moderation_reports WHERE status IN ('open','investigating')`},
		{"Overdue support", "Open tickets beyond the response target", "/admin/support", `SELECT count(*) FROM admin_support_tickets WHERE status IN ('open','in_progress') AND created_at < CURRENT_TIMESTAMP - (SELECT value::integer FROM admin_platform_settings WHERE key='support_response_hours') * INTERVAL '1 hour'`},
	}
	var result []models.AdminMetric
	for _, q := range queries {
		var n int
		if err := r.db.QueryRowContext(ctx, q.query).Scan(&n); err != nil {
			return nil, err
		}
		path := q.path
		if !user.CanAdmin("marketplace") && q.label == "Listings" {
			path = ""
		}
		result = append(result, models.AdminMetric{Label: q.label, Value: strconv.Itoa(n), Detail: q.detail, Path: path})
	}
	if user.CanAdmin("finance") {
		var value string
		if err := r.db.QueryRowContext(ctx, `SELECT COALESCE(string_agg(currency || ' ' || total::text, ' · ' ORDER BY currency),'0.00') FROM (SELECT currency,SUM(amount) AS total FROM platform_revenue GROUP BY currency) revenue`).Scan(&value); err != nil {
			return nil, err
		}
		result = append(result, models.AdminMetric{Label: "Revenue", Value: value, Detail: "Recorded platform commissions", Path: "/admin/finance"})
		var n int
		if err := r.db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM wallet_withdrawals WHERE status IN ('pending','processing'))+(SELECT count(*) FROM wallet_transfers WHERE status='pending')`).Scan(&n); err != nil {
			return nil, err
		}
		result = append(result, models.AdminMetric{Label: "Payment requests", Value: strconv.Itoa(n), Detail: "Withdrawals and transfers awaiting review", Path: "/admin/finance"})
	}
	return result, nil
}

// Every search and page value is bound. Only these fixed, reviewed queries run.
func adminQueries(section string) []adminQuery {
	users := adminQuery{"users", "Member accounts", `SELECT u.id::text,u.account_status,u.id::text,u.full_name,u.phone,u.role,u.account_status,COALESCE(v.verification_level,'unverified'),to_char(u.created_at,'YYYY-MM-DD') FROM users u LEFT JOIN user_verifications v ON v.user_id=u.id WHERE u.deleted_at IS NULL AND ($1='' OR concat_ws(' ',u.id,u.full_name,u.phone,u.role,u.account_status,v.verification_level) ILIKE '%'||$1||'%') ORDER BY u.created_at DESC,u.id DESC LIMIT 26 OFFSET $2`, []string{"ID", "Name", "Phone", "Member role", "Account status", "Verification", "Joined"}}
	reports := adminQuery{"moderation", "Account & listing reports", `SELECT r.id::text,r.status,r.id::text,CASE WHEN r.user_id IS NOT NULL THEN 'Account #'||r.user_id ELSE 'Listing #'||r.product_id END,r.reason,r.status,r.resolution,COALESCE(u.full_name,'Unassigned'),to_char(r.updated_at,'YYYY-MM-DD HH24:MI') FROM admin_moderation_reports r LEFT JOIN users u ON u.id=r.reviewed_by WHERE ($1='' OR concat_ws(' ',r.id,r.user_id,r.product_id,r.reason,r.status) ILIKE '%'||$1||'%') ORDER BY r.updated_at DESC,r.id DESC LIMIT 26 OFFSET $2`, []string{"ID", "Reported item", "Reason", "Status", "Resolution", "Reviewed by", "Updated"}}
	tickets := adminQuery{"tickets", "Support tickets, complaints & disputes", `SELECT t.id::text,t.status,t.id::text,u.full_name||' (#'||u.id||')',t.subject,t.description,t.kind,COALESCE(t.order_id::text,'—'),t.priority,t.status,COALESCE(s.full_name,'Unassigned'),t.resolution,to_char(t.updated_at,'YYYY-MM-DD HH24:MI') FROM admin_support_tickets t JOIN users u ON u.id=t.user_id LEFT JOIN users s ON s.id=t.assigned_to WHERE ($1='' OR concat_ws(' ',t.id,t.user_id,t.order_id,u.full_name,t.subject,t.kind,t.status) ILIKE '%'||$1||'%') ORDER BY CASE t.priority WHEN 'urgent' THEN 0 WHEN 'high' THEN 1 ELSE 2 END,t.updated_at DESC,t.id DESC LIMIT 26 OFFSET $2`, []string{"ID", "Member", "Subject", "Description", "Type", "Order", "Priority", "Status", "Assigned staff", "Resolution", "Updated"}}
	orders := adminQuery{"orders", "Marketplace orders", `SELECT o.id::text,o.status,o.id::text,u.full_name,o.status,o.total_amount::text,o.currency,to_char(o.created_at,'YYYY-MM-DD HH24:MI') FROM orders o JOIN users u ON u.id=o.buyer_id WHERE ($1='' OR concat_ws(' ',o.id,u.full_name,o.status) ILIKE '%'||$1||'%') ORDER BY o.created_at DESC,o.id DESC LIMIT 26 OFFSET $2`, []string{"Order", "Buyer", "Status", "Amount", "Currency", "Created"}}
	switch section {
	case "overview":
		return []adminQuery{tickets, reports}
	case "users":
		return []adminQuery{users, reports}
	case "marketplace":
		return []adminQuery{
			{"products", "Products & listings", `SELECT p.id::text,p.status,p.id::text,p.name,p.category,COALESCE(u.full_name,'Organization #'||p.owner_id),p.quantity::text||' '||p.unit,p.price::text,p.status,to_char(p.created_at,'YYYY-MM-DD') FROM products p LEFT JOIN users u ON p.owner_type='user' AND u.id=p.owner_id WHERE ($1='' OR concat_ws(' ',p.id,p.name,p.category,u.full_name,p.status) ILIKE '%'||$1||'%') ORDER BY p.created_at DESC,p.id DESC LIMIT 26 OFFSET $2`, []string{"ID", "Product", "Category", "Seller", "Available stock", "Unit price", "Status", "Listed"}},
			{"categories", "Marketplace categories", `SELECT category,'',category,count(*)::text,count(*) FILTER(WHERE status='active')::text FROM products WHERE ($1='' OR category ILIKE '%'||$1||'%') GROUP BY category ORDER BY category LIMIT 26 OFFSET $2`, []string{"Category", "Listings", "Active listings"}}, reports,
			{"price_reports", "Market price reports & suspicious activity", `SELECT id::text,status,id::text,commodity_id,market_id,price::text||' '||currency||' / '||unit,status,flag_reason,to_char(submitted_at,'YYYY-MM-DD HH24:MI') FROM price_reports WHERE ($1='' OR concat_ws(' ',commodity_id,market_id,status,flag_reason) ILIKE '%'||$1||'%') ORDER BY submitted_at DESC,id LIMIT 26 OFFSET $2`, []string{"Report", "Commodity", "Market", "Price", "Status", "Flag reason", "Submitted"}},
		}
	case "orders":
		return []adminQuery{orders,
			{"deliveries", "Delivery tracking", `SELECT d.id::text,d.status,d.id::text,d.order_id::text,d.status,COALESCE(p.name,'Unassigned'),COALESCE((SELECT dt.location_name FROM delivery_tracking dt WHERE dt.delivery_id=d.id ORDER BY dt.created_at DESC,dt.id DESC LIMIT 1),'No tracking update'),to_char(d.updated_at,'YYYY-MM-DD HH24:MI') FROM deliveries d LEFT JOIN delivery_partners p ON p.id=d.delivery_partner_id WHERE ($1='' OR concat_ws(' ',d.id,d.order_id,d.status,p.name) ILIKE '%'||$1||'%') ORDER BY d.updated_at DESC,d.id DESC LIMIT 26 OFFSET $2`, []string{"Delivery", "Order", "Status", "Delivery partner", "Last known location", "Updated"}}, tickets}
	case "support":
		return []adminQuery{tickets,
			{"ticket_history", "Resolution history", `SELECT h.id::text,h.status,h.ticket_id::text,u.full_name,h.status,h.note,to_char(h.created_at,'YYYY-MM-DD HH24:MI') FROM admin_ticket_history h JOIN users u ON u.id=h.actor_id WHERE ($1='' OR concat_ws(' ',h.ticket_id,u.full_name,h.note) ILIKE '%'||$1||'%') ORDER BY h.created_at DESC,h.id DESC LIMIT 26 OFFSET $2`, []string{"Ticket", "Staff member", "Status", "Note", "When"}}}
	case "staff":
		return []adminQuery{
			{"staff", "Staff accounts", `SELECT id::text,CASE WHEN staff_disabled THEN 'disabled' ELSE 'enabled' END,id::text,full_name,phone,COALESCE(email,''),CASE WHEN staff_role='' AND is_admin THEN 'super_admin' ELSE staff_role END,CASE WHEN staff_disabled THEN 'disabled' ELSE account_status END,to_char(updated_at,'YYYY-MM-DD HH24:MI') FROM users WHERE deleted_at IS NULL AND (staff_role<>'' OR is_admin) AND ($1='' OR concat_ws(' ',id,full_name,phone,staff_role) ILIKE '%'||$1||'%') ORDER BY id LIMIT 26 OFFSET $2`, []string{"ID", "Name", "Phone", "Email", "Staff role", "Access", "Updated"}}}
	case "audit":
		return []adminQuery{
			{"audit", "Administrative action history", `SELECT id::text,CASE WHEN succeeded THEN 'succeeded' ELSE 'failed' END,id::text,actor_name,actor_role,action,target,before_value::text,after_value::text,CASE WHEN succeeded THEN 'Succeeded' ELSE 'Failed' END,detail,to_char(created_at,'YYYY-MM-DD HH24:MI:SS') FROM admin_audit_logs WHERE ($1='' OR concat_ws(' ',actor_name,action,target,detail) ILIKE '%'||$1||'%') ORDER BY created_at DESC,id DESC LIMIT 26 OFFSET $2`, []string{"ID", "Actor", "Role", "Action", "Target", "Before", "After", "Result", "Detail", "When (UTC)"}}}
	case "finance":
		return adminFinanceQueries()
	case "reports":
		return adminAnalyticsQueries()
	}
	return nil
}

func adminFinanceQueries() []adminQuery {
	return []adminQuery{
		{"wallets", "Wallet balances", `SELECT w.id::text,'',w.id::text,COALESCE(u.full_name,'Organization #'||w.owner_id),w.wallet_number,w.currency,w.balance::text,to_char(w.updated_at,'YYYY-MM-DD HH24:MI') FROM wallets w LEFT JOIN users u ON w.owner_type='user' AND u.id=w.owner_id WHERE ($1='' OR concat_ws(' ',w.id,u.full_name,w.wallet_number) ILIKE '%'||$1||'%') ORDER BY w.id DESC LIMIT 26 OFFSET $2`, []string{"Wallet", "Owner", "Wallet number", "Currency", "Balance", "Updated"}},
		{"deposits", "Deposits", `SELECT d.id::text,d.status,d.reference,COALESCE(u.full_name,'Organization #'||w.owner_id),d.currency,d.amount::text,d.status,to_char(d.created_at,'YYYY-MM-DD HH24:MI') FROM wallet_deposits d JOIN wallets w ON w.id=d.wallet_id LEFT JOIN users u ON w.owner_type='user' AND u.id=w.owner_id WHERE ($1='' OR concat_ws(' ',d.reference,u.full_name,d.status) ILIKE '%'||$1||'%') ORDER BY d.created_at DESC,d.id DESC LIMIT 26 OFFSET $2`, []string{"Reference", "Member", "Currency", "Amount", "Status", "Created"}},
		{"withdrawals", "Withdrawals", `SELECT d.id::text,d.status,d.id::text,d.reference,u.full_name,d.currency,d.amount::text,d.status,COALESCE(s.full_name,'Awaiting approval'),to_char(d.updated_at,'YYYY-MM-DD HH24:MI') FROM wallet_withdrawals d JOIN users u ON u.id=d.user_id LEFT JOIN users s ON s.id=d.approved_by WHERE ($1='' OR concat_ws(' ',d.reference,u.full_name,d.status) ILIKE '%'||$1||'%') ORDER BY d.created_at DESC,d.id DESC LIMIT 26 OFFSET $2`, []string{"ID", "Reference", "Member", "Currency", "Amount", "Status", "Approved by", "Updated"}},
		{"transfers", "Wallet transfers", `SELECT t.id::text,t.status,t.id::text,t.reference,s.full_name,r.full_name,t.currency,t.amount::text,t.status,to_char(t.updated_at,'YYYY-MM-DD HH24:MI') FROM wallet_transfers t JOIN users s ON s.id=t.sender_id JOIN users r ON r.id=t.recipient_id WHERE ($1='' OR concat_ws(' ',t.reference,s.full_name,r.full_name,t.status) ILIKE '%'||$1||'%') ORDER BY t.created_at DESC,t.id DESC LIMIT 26 OFFSET $2`, []string{"ID", "Reference", "Sender", "Recipient", "Currency", "Amount", "Status", "Updated"}},
		{"commissions", "Platform commissions", `SELECT id::text,'',order_id::text,reference,currency,gross_amount::text,rate_percent::text||'%',amount::text,to_char(created_at,'YYYY-MM-DD HH24:MI') FROM platform_revenue WHERE ($1='' OR concat_ws(' ',order_id,reference,currency) ILIKE '%'||$1||'%') ORDER BY created_at DESC,id DESC LIMIT 26 OFFSET $2`, []string{"Order", "Reference", "Currency", "Gross payment", "Rate", "Commission", "Recorded"}},
		{"transactions", "Payments & recorded refunds", `SELECT t.id::text,t.status,t.order_id::text,COALESCE(t.payment_reference,''),u.full_name,t.currency,t.amount::text,t.status,CASE WHEN t.review_required THEN 'Needs investigation / possible refund' ELSE '—' END,to_char(t.updated_at,'YYYY-MM-DD HH24:MI') FROM transactions t JOIN users u ON u.id=t.buyer_id WHERE ($1='' OR concat_ws(' ',t.order_id,t.payment_reference,u.full_name,t.status) ILIKE '%'||$1||'%') ORDER BY t.updated_at DESC,t.id DESC LIMIT 26 OFFSET $2`, []string{"Order", "Reference", "Buyer", "Currency", "Amount", "Status", "Review required", "Updated"}},
		{"settlements", "Settlement & wallet ledger records", `SELECT l.id::text,'',l.id::text,l.wallet_id::text,l.type,l.amount::text,w.currency,COALESCE(l.reference_type,''),COALESCE(l.reference_id::text,''),COALESCE(l.description,''),to_char(l.created_at,'YYYY-MM-DD HH24:MI') FROM wallet_transactions l JOIN wallets w ON w.id=l.wallet_id WHERE ($1='' OR concat_ws(' ',l.wallet_id,l.reference_type,l.reference_id,l.description) ILIKE '%'||$1||'%') ORDER BY l.created_at DESC,l.id DESC LIMIT 26 OFFSET $2`, []string{"Entry", "Wallet", "Direction", "Amount", "Currency", "Reference type", "Reference", "Description", "Recorded"}},
	}
}

func adminAnalyticsQueries() []adminQuery {
	return []adminQuery{
		{"growth", "Monthly member & farmer growth", `SELECT to_char(created_at,'YYYY-MM'),' ',to_char(created_at,'YYYY-MM'),count(*)::text,count(*) FILTER(WHERE role='farmer')::text,count(*) FILTER(WHERE role='buyer')::text FROM users WHERE deleted_at IS NULL AND ($1='' OR to_char(created_at,'YYYY-MM') ILIKE '%'||$1||'%') GROUP BY 1 ORDER BY 1 DESC LIMIT 26 OFFSET $2`, []string{"Month", "New members", "Farmers", "Buyers"}},
		{"sales_summary", "Sales & transaction summaries", `SELECT to_char(created_at,'YYYY-MM')||currency,'',to_char(created_at,'YYYY-MM'),currency,count(*)::text,COALESCE(SUM(amount) FILTER(WHERE status='successful'),0)::text,count(*) FILTER(WHERE status='failed')::text,COALESCE(SUM(amount) FILTER(WHERE status='refunded'),0)::text FROM transactions WHERE ($1='' OR concat_ws(' ',currency,to_char(created_at,'YYYY-MM')) ILIKE '%'||$1||'%') GROUP BY 1,currency,to_char(created_at,'YYYY-MM') ORDER BY 1 DESC LIMIT 26 OFFSET $2`, []string{"Month", "Currency", "Payment attempts", "Successful sales", "Failed attempts", "Recorded refunds"}},
		{"activity", "Marketplace activity by category", `SELECT category,'',category,count(*)::text,count(*) FILTER(WHERE status='active')::text,count(*) FILTER(WHERE status='sold_out')::text FROM products WHERE ($1='' OR category ILIKE '%'||$1||'%') GROUP BY category ORDER BY count(*) DESC,category LIMIT 26 OFFSET $2`, []string{"Category", "Listings", "Active", "Sold out"}},
		{"performance", "Support operational performance", `SELECT COALESCE(s.id::text,'unassigned'),'',COALESCE(s.full_name,'Unassigned'),count(*)::text,count(*) FILTER(WHERE t.status IN ('open','in_progress'))::text,count(*) FILTER(WHERE t.status IN ('resolved','closed'))::text,COALESCE(round((AVG(EXTRACT(EPOCH FROM (t.updated_at-t.created_at))/3600) FILTER(WHERE t.status IN ('resolved','closed')))::numeric,1)::text,'—') FROM admin_support_tickets t LEFT JOIN users s ON s.id=t.assigned_to WHERE ($1='' OR COALESCE(s.full_name,'Unassigned') ILIKE '%'||$1||'%') GROUP BY s.id,s.full_name ORDER BY count(*) DESC,1 LIMIT 26 OFFSET $2`, []string{"Staff", "Tickets", "Open", "Resolved / closed", "Average hours to last resolution update"}},
		{"delivery_performance", "Delivery operational performance", `SELECT status,'',status,count(*)::text FROM deliveries WHERE ($1='' OR status ILIKE '%'||$1||'%') GROUP BY status ORDER BY status LIMIT 26 OFFSET $2`, []string{"Delivery status", "Deliveries"}},
	}
}
