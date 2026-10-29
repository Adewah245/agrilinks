package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/mail"
	"strconv"
	"strings"

	"backend/internal/models"
)

var ErrAdminForbidden = errors.New("you do not have permission to perform this action")
var ErrAdminInvalid = errors.New("invalid admin request")

func adminInvalid(message string) error { return fmt.Errorf("%w: %s", ErrAdminInvalid, message) }

var adminSettingLabels = map[string]string{
	"support_default_priority":     "Default support priority",
	"support_response_hours":       "Support response target (hours)",
	"dispute_policy":               "Dispute resolution policy",
	"listing_policy":               "Listing review policy",
	"notification_ticket_opened":   "Ticket opened notification",
	"notification_ticket_resolved": "Ticket resolved notification",
}

func AdminActionSection(action string) string {
	switch action {
	case "user_status", "user_verification", "product_status", "report_update":
		return "marketplace"
	case "report_create", "ticket_create", "ticket_update":
		return "support"
	case "staff_create", "staff_role", "staff_access":
		return "staff"
	case "setting_update":
		return "settings"
	}
	return ""
}

func (r *AdminRepository) Audit(ctx context.Context, actor *models.User, action, target string, before, after string, success bool, detail string) error {
	if before == "" {
		before = "{}"
	}
	if after == "" {
		after = "{}"
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO admin_audit_logs(actor_id,actor_name,actor_role,action,target,before_value,after_value,succeeded,detail) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9)`, actor.ID, actor.FullName, actor.AdminRole(), action, target, before, after, success, detail)
	return err
}

func (r *AdminRepository) Apply(ctx context.Context, actor *models.User, fields map[string]string) error {
	action := fields["action"]
	section := AdminActionSection(action)
	if !actor.CanAdmin(section) {
		return ErrAdminForbidden
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Serialize staff permission changes, including protection of the last Super Admin.
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('agro-shield:admin-actions',0))`); err != nil {
		return err
	}
	var current models.User
	if err := tx.QueryRowContext(ctx, `SELECT id,full_name,is_admin,staff_role,staff_disabled,account_status FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, actor.ID).Scan(&current.ID, &current.FullName, &current.IsAdmin, &current.StaffRole, &current.StaffDisabled, &current.AccountStatus); err != nil {
		return err
	}
	if actor.IsAdmin && current.StaffRole == "" {
		current.IsAdmin = true
	}
	if !current.CanAdmin(section) {
		return ErrAdminForbidden
	}
	actor = &current
	before, after, target, err := applyAdminAction(ctx, tx, actor, fields)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO admin_audit_logs(actor_id,actor_name,actor_role,action,target,before_value,after_value,succeeded,detail) VALUES($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,TRUE,$8)`, actor.ID, actor.FullName, actor.AdminRole(), action, target, before, after, fields["reason"]); err != nil {
		return err
	}
	return tx.Commit()
}

func applyAdminAction(ctx context.Context, tx *sql.Tx, actor *models.User, f map[string]string) (string, string, string, error) {
	action := f["action"]
	id, _ := strconv.Atoi(f["id"])
	reason := strings.TrimSpace(f["reason"])
	before, after, target := "{}", "{}", action+":"+f["id"]
	if len(reason) > 2000 {
		return before, after, target, adminInvalid("reason is too long")
	}
	if action != "staff_create" && action != "ticket_create" && action != "report_create" && reason == "" {
		return before, after, target, adminInvalid("record a reason for the change")
	}
	var err error
	switch action {
	case "staff_create":
		role := f["role"]
		phone := f["phone"]
		name := strings.TrimSpace(f["name"])
		email := strings.TrimSpace(f["email"])
		if !validStaffRole(role) || name == "" || len(name) > 200 || phone == "" || f["password_hash"] == "" {
			return before, after, target, adminInvalid("name, phone, password and a valid staff role are required")
		}
		if email != "" {
			address, e := mail.ParseAddress(email)
			if e != nil || address.Address != email {
				return before, after, target, adminInvalid("enter a valid email address")
			}
		}
		err = tx.QueryRowContext(ctx, `INSERT INTO users(full_name,phone,email,password_hash,role,account_type,staff_role,photo_url) VALUES($1,$2,NULLIF($3,''),$4,'buyer','individual',$5,$6) RETURNING id`, name, phone, email, f["password_hash"], role, defaultPhotoURL).Scan(&id)
		if err != nil {
			return before, after, target, mapUserUniqueViolation(err)
		}
		target = "user:" + strconv.Itoa(id)
		after, err = adminUserSnapshot(ctx, tx, id)
	case "staff_role", "staff_access", "user_status":
		before, err = adminUserSnapshot(ctx, tx, id)
		if err != nil {
			break
		}
		target = "user:" + strconv.Itoa(id)
		var role string
		var isAdmin, disabled bool
		if err = tx.QueryRowContext(ctx, `SELECT staff_role,is_admin,staff_disabled FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&role, &isAdmin, &disabled); err != nil {
			break
		}
		if action == "user_status" {
			if role != "" || isAdmin || id == actor.ID {
				return before, after, target, adminInvalid("manage staff accounts in Staff, Roles & Permissions")
			}
			if f["value"] != "active" && f["value"] != "suspended" {
				return before, after, target, adminInvalid("choose an account status")
			}
			_, err = tx.ExecContext(ctx, `UPDATE users SET account_status=$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, f["value"])
		} else {
			if id == actor.ID {
				return before, after, target, adminInvalid("another Super Admin must change your own staff access")
			}
			if action == "staff_role" && !validStaffRole(f["role"]) {
				return before, after, target, adminInvalid("choose a valid staff role")
			}
			if action == "staff_role" {
				var status string
				if err = tx.QueryRowContext(ctx, `SELECT account_status FROM users WHERE id=$1`, id).Scan(&status); err != nil {
					break
				}
				if status != "active" {
					return before, after, target, adminInvalid("activate the member account before assigning a staff role")
				}
			}
			if action == "staff_access" && f["value"] != "enabled" && f["value"] != "disabled" {
				return before, after, target, adminInvalid("choose enabled or disabled")
			}
			if action == "staff_access" && role == "" && !isAdmin {
				return before, after, target, adminInvalid("this account is not a staff member")
			}
			if (role == "super_admin" || role == "" && isAdmin) && !disabled && (action == "staff_role" && f["role"] != "super_admin" || action == "staff_access" && f["value"] == "disabled") {
				var count int
				if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM users WHERE deleted_at IS NULL AND NOT staff_disabled AND account_status='active' AND (staff_role='super_admin' OR (staff_role='' AND is_admin))`).Scan(&count); err != nil {
					break
				}
				if count <= 1 {
					return before, after, target, adminInvalid("keep at least one enabled Super Admin")
				}
			}
			if action == "staff_role" {
				_, err = tx.ExecContext(ctx, `UPDATE users SET staff_role=$2,is_admin=FALSE,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, f["role"])
			} else {
				_, err = tx.ExecContext(ctx, `UPDATE users SET staff_disabled=$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, f["value"] == "disabled")
			}
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id=$1`, id)
		}
		if err == nil {
			after, err = adminUserSnapshot(ctx, tx, id)
		}
	case "user_verification":
		value := f["value"]
		if value != "pending" && value != "verified" && value != "rejected" && value != "unverified" {
			return before, after, target, adminInvalid("choose a verification status")
		}
		var staff string
		if err = tx.QueryRowContext(ctx, `SELECT staff_role FROM users WHERE id=$1 AND deleted_at IS NULL`, id).Scan(&staff); err != nil {
			break
		}
		target = "verification:" + strconv.Itoa(id)
		if err = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('verification',COALESCE((SELECT verification_level FROM user_verifications WHERE user_id=$1),'unverified'))::text`, id).Scan(&before); err != nil {
			break
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO user_verifications(user_id,verification_level) VALUES($1,$2) ON CONFLICT(user_id) DO UPDATE SET verification_level=$2`, id, value)
		if err == nil {
			err = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('verification',verification_level)::text FROM user_verifications WHERE user_id=$1`, id).Scan(&after)
		}
	case "product_status":
		value := f["value"]
		var old string
		var quantity float64
		if value != "active" && value != "inactive" {
			return before, after, target, adminInvalid("choose active or inactive")
		}
		if err = tx.QueryRowContext(ctx, `SELECT status,quantity FROM products WHERE id=$1 FOR UPDATE`, id).Scan(&old, &quantity); err != nil {
			break
		}
		if value == "active" && quantity <= 0 {
			return before, after, target, adminInvalid("a listing needs stock before it can be activated")
		}
		target = "product:" + strconv.Itoa(id)
		if err = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('status',status)::text FROM products WHERE id=$1`, id).Scan(&before); err != nil {
			break
		}
		_, err = tx.ExecContext(ctx, `UPDATE products SET status=$2,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, value)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO product_status_history(product_id,old_status,new_status,reason,changed_by_type,changed_by_id) VALUES($1,$2,$3,$4,'user',$5)`, id, old, value, reason, actor.ID)
		}
		if err == nil {
			err = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('status',status)::text FROM products WHERE id=$1`, id).Scan(&after)
		}
	case "report_create":
		kind := f["target_kind"]
		targetID, _ := strconv.Atoi(f["target_id"])
		if targetID <= 0 || reason == "" {
			return before, after, target, adminInvalid("select an account or listing and record the report reason")
		}
		if kind == "product" && !actor.CanAdmin("marketplace") {
			return before, after, target, ErrAdminForbidden
		}
		if kind == "user" {
			err = tx.QueryRowContext(ctx, `INSERT INTO admin_moderation_reports(user_id,reason,created_by) VALUES($1,$2,$3) RETURNING id`, targetID, reason, actor.ID).Scan(&id)
		} else if kind == "product" {
			err = tx.QueryRowContext(ctx, `INSERT INTO admin_moderation_reports(product_id,reason,created_by) VALUES($1,$2,$3) RETURNING id`, targetID, reason, actor.ID).Scan(&id)
		} else {
			return before, after, target, adminInvalid("choose account or listing")
		}
		target = "report:" + strconv.Itoa(id)
		if err == nil {
			after, err = adminReportSnapshot(ctx, tx, id)
		}
	case "report_update":
		value := f["value"]
		if value != "open" && value != "investigating" && value != "resolved" && value != "dismissed" {
			return before, after, target, adminInvalid("choose a report status")
		}
		target = "report:" + strconv.Itoa(id)
		before, err = adminReportSnapshot(ctx, tx, id)
		if err != nil {
			break
		}
		_, err = tx.ExecContext(ctx, `UPDATE admin_moderation_reports SET status=$2,resolution=$3,reviewed_by=$4,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, value, reason, actor.ID)
		if err == nil {
			after, err = adminReportSnapshot(ctx, tx, id)
		}
	case "ticket_create":
		memberID, _ := strconv.Atoi(f["member_id"])
		orderID, _ := strconv.Atoi(f["order_id"])
		subject := strings.TrimSpace(f["subject"])
		description := strings.TrimSpace(f["description"])
		kind := f["kind"]
		priority := f["priority"]
		if memberID <= 0 || subject == "" || len(subject) > 200 || description == "" || len(description) > 4000 {
			return before, after, target, adminInvalid("member ID, subject and description are required")
		}
		if kind != "support" && kind != "complaint" && kind != "dispute" && kind != "refund" {
			return before, after, target, adminInvalid("choose a ticket type")
		}
		if priority == "" {
			if err = tx.QueryRowContext(ctx, `SELECT value FROM admin_platform_settings WHERE key='support_default_priority'`).Scan(&priority); err != nil {
				break
			}
		}
		if priority != "normal" && priority != "high" && priority != "urgent" {
			return before, after, target, adminInvalid("choose a ticket priority")
		}
		var order any
		if orderID > 0 {
			var exists bool
			if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM orders WHERE id=$1 AND (buyer_id=$2 OR EXISTS(SELECT 1 FROM order_items i JOIN products p ON p.id=i.product_id WHERE i.order_id=$1 AND p.owner_type='user' AND p.owner_id=$2)))`, orderID, memberID).Scan(&exists); err != nil {
				break
			}
			if !exists {
				return before, after, target, adminInvalid("the order must belong to the ticket's buyer or seller")
			}
			order = orderID
		}
		err = tx.QueryRowContext(ctx, `INSERT INTO admin_support_tickets(user_id,order_id,subject,description,kind,priority,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, memberID, order, subject, description, kind, priority, actor.ID).Scan(&id)
		target = "ticket:" + strconv.Itoa(id)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO admin_ticket_history(ticket_id,actor_id,status,note) VALUES($1,$2,'open',$3)`, id, actor.ID, description)
		}
		if err == nil {
			after, err = adminTicketSnapshot(ctx, tx, id)
		}
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,notification_type,title,message) SELECT $1,'system','Support request opened',value||' Ticket #'||$2::bigint::text FROM admin_platform_settings WHERE key='notification_ticket_opened'`, memberID, id)
		}
	case "ticket_update":
		value := f["value"]
		assignee := f["assignee"]
		if value != "open" && value != "in_progress" && value != "resolved" && value != "closed" {
			return before, after, target, adminInvalid("choose a ticket status")
		}
		target = "ticket:" + strconv.Itoa(id)
		before, err = adminTicketSnapshot(ctx, tx, id)
		if err != nil {
			break
		}
		var assigned any
		if assignee != "" && assignee != "keep" {
			n, e := strconv.Atoi(assignee)
			if e != nil || n <= 0 {
				return before, after, target, adminInvalid("choose a staff member")
			}
			var enabled bool
			if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE id=$1 AND deleted_at IS NULL AND NOT staff_disabled AND account_status='active' AND (staff_role<>'' OR is_admin))`, n).Scan(&enabled); err != nil {
				break
			}
			if !enabled {
				return before, after, target, adminInvalid("assign tickets to enabled staff accounts")
			}
			assigned = n
		}
		_, err = tx.ExecContext(ctx, `UPDATE admin_support_tickets SET status=$2,assigned_to=CASE WHEN $3='keep' THEN assigned_to ELSE $4::integer END,resolution=CASE WHEN $2 IN ('resolved','closed') THEN $5 ELSE resolution END,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, id, value, assignee, assigned, reason)
		if err == nil {
			_, err = tx.ExecContext(ctx, `INSERT INTO admin_ticket_history(ticket_id,actor_id,status,note) VALUES($1,$2,$3,$4)`, id, actor.ID, value, reason)
		}
		if err == nil {
			after, err = adminTicketSnapshot(ctx, tx, id)
		}
		if err == nil && (value == "resolved" || value == "closed") {
			_, err = tx.ExecContext(ctx, `INSERT INTO notifications(user_id,notification_type,title,message) SELECT t.user_id,'system','Support request resolved',s.value||' Ticket #'||t.id::text FROM admin_support_tickets t JOIN admin_platform_settings s ON s.key='notification_ticket_resolved' WHERE t.id=$1`, id)
		}
	case "setting_update":
		key := f["key"]
		value := strings.TrimSpace(f["value"])
		if adminSettingLabels[key] == "" || value == "" || len(value) > 4000 {
			return before, after, target, adminInvalid("choose an approved setting and provide a value")
		}
		if key == "support_default_priority" && value != "normal" && value != "high" && value != "urgent" {
			return before, after, target, adminInvalid("default priority must be normal, high or urgent")
		}
		if key == "support_response_hours" {
			n, e := strconv.Atoi(value)
			if e != nil || n < 1 || n > 720 {
				return before, after, target, adminInvalid("response target must be between 1 and 720 hours")
			}
		}
		target = "setting:" + key
		if err = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('value',value)::text FROM admin_platform_settings WHERE key=$1 FOR UPDATE`, key).Scan(&before); err != nil {
			break
		}
		_, err = tx.ExecContext(ctx, `UPDATE admin_platform_settings SET value=$2,updated_by=$3,updated_at=CURRENT_TIMESTAMP WHERE key=$1`, key, value, actor.ID)
		if err == nil {
			err = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('value',value)::text FROM admin_platform_settings WHERE key=$1`, key).Scan(&after)
		}
	default:
		return before, after, target, adminInvalid("choose a supported action")
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = adminInvalid("the requested record was not found")
	}
	return before, after, target, err
}

func validStaffRole(role string) bool {
	return role == "super_admin" || role == "customer_care" || role == "operations_manager"
}
func adminUserSnapshot(ctx context.Context, tx *sql.Tx, id int) (string, error) {
	var value string
	err := tx.QueryRowContext(ctx, `SELECT jsonb_build_object('name',full_name,'staff_role',staff_role,'staff_disabled',staff_disabled,'account_status',account_status)::text FROM users WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, id).Scan(&value)
	return value, err
}
func adminReportSnapshot(ctx context.Context, tx *sql.Tx, id int) (string, error) {
	var value string
	err := tx.QueryRowContext(ctx, `SELECT jsonb_build_object('user_id',user_id,'product_id',product_id,'status',status,'reason',reason,'resolution',resolution,'reviewed_by',reviewed_by)::text FROM admin_moderation_reports WHERE id=$1 FOR UPDATE`, id).Scan(&value)
	return value, err
}
func adminTicketSnapshot(ctx context.Context, tx *sql.Tx, id int) (string, error) {
	var value string
	err := tx.QueryRowContext(ctx, `SELECT jsonb_build_object('member_id',user_id,'order_id',order_id,'subject',subject,'priority',priority,'status',status,'assigned_to',assigned_to,'resolution',resolution)::text FROM admin_support_tickets WHERE id=$1 FOR UPDATE`, id).Scan(&value)
	return value, err
}

// Bootstrap promotes an existing member only while no enabled Super Admin exists.
// It is exposed by a local CLI, never by an unauthenticated web route.
func (r *AdminRepository) Bootstrap(ctx context.Context, phone string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('agro-shield:admin-actions',0))`); err != nil {
		return err
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE deleted_at IS NULL AND NOT staff_disabled AND account_status='active' AND (staff_role='super_admin' OR (staff_role='' AND is_admin)))`).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return adminInvalid("an enabled Super Admin already exists; use the staff dashboard")
	}
	var actor models.User
	if err = tx.QueryRowContext(ctx, `SELECT id,full_name FROM users WHERE phone=$1 AND deleted_at IS NULL AND account_status='active' AND NOT staff_disabled FOR UPDATE`, phone).Scan(&actor.ID, &actor.FullName); err != nil {
		return fmt.Errorf("find enabled account: %w", err)
	}
	before, err := adminUserSnapshot(ctx, tx, actor.ID)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE users SET staff_role='super_admin',is_admin=FALSE,updated_at=CURRENT_TIMESTAMP WHERE id=$1`, actor.ID); err != nil {
		return err
	}
	after, err := adminUserSnapshot(ctx, tx, actor.ID)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id=$1`, actor.ID); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO admin_audit_logs(actor_id,actor_name,actor_role,action,target,before_value,after_value,succeeded,detail) VALUES($1,$2,'super_admin','bootstrap',$3,$4::jsonb,$5::jsonb,TRUE,'First Super Admin created by local CLI')`, actor.ID, actor.FullName, "user:"+strconv.Itoa(actor.ID), before, after); err != nil {
		return err
	}
	return tx.Commit()
}
