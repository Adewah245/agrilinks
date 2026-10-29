# Admin workspace

The central staff dashboard is `/admin`. Staff sign in through `/login`; their
login opens the admin overview. `/wallet/admin` redirects authorized operators
to `/admin/finance`.

## First Super Admin

Apply migration `065_admin_workspace.sql` using the normal migration command or
server startup. Existing `users.is_admin` accounts become Super Admins. Existing
`WALLET_ADMIN_USER_IDS` configuration remains supported; explicit staff roles
and disabled/suspended status take precedence over that legacy configuration.

For a new installation, register the intended owner normally, then run from
`backend` with the application's database configuration:

```sh
go run ./cmd/admin -phone 08012345678
```

Replace the example phone with the registered owner's number. The command
promotes an enabled existing account, revokes its previous sessions, and records
the bootstrap action. It refuses to run when an enabled persisted Super Admin
already exists. It is a local command; there is no public setup endpoint.

Log in again, open `/admin/staff`, and either create a staff account or assign a
staff role to an existing member ID. Staff roles are independent of farmer,
buyer, and learner roles. Staff accounts created in the dashboard use buyer as
their member role and sign in with their phone and password.

## Permissions

| Section | Path | Super Admin | Customer Care | Operations Manager |
| --- | --- | --- | --- | --- |
| Dashboard Overview | `/admin` | Yes | Yes | Yes |
| Users & Farmer Management | `/admin/users` | Manage | View and report | Manage |
| Marketplace Management | `/admin/marketplace` | Yes | No | Yes |
| Orders & Deliveries | `/admin/orders` | Yes | Yes | Yes |
| Finance & Payments | `/admin/finance` | Yes | No | Yes |
| Customer Care & Disputes | `/admin/support` | Yes | Yes | Yes |
| Reports & Analytics | `/admin/reports` | Yes | No | Yes |
| Audit Logs | `/admin/audit` | Yes | No | No |
| Staff, Roles & Permissions | `/admin/staff` | Yes | No | No |
| Platform Settings | `/admin/settings` | Yes | No | No |

The backend checks every section and action. Navigation also reflects the
current staff role. Customer Care cannot change account status or verification,
moderate listings, approve money movements, change settings, or grant staff
permissions. Its overview omits revenue and payment request metrics.

Staff role changes revoke existing sessions. Disabling staff or suspending a
member blocks login and authenticated requests and revokes sessions. Staff
cannot change their own role/access; another Super Admin must make that change.
The last enabled persisted Super Admin cannot be demoted or disabled. Staff
identities cannot be closed through member account deletion, preserving their
relationship to administrative history.

## Records and workflows

Overview cards use saved PostgreSQL records. Active farmers means farmer
accounts that are enabled and not closed. Revenue is the recorded platform
commission, grouped by currency. Pending tasks include open support tickets
and pending verifications; payment requests appear separately for finance
roles. Overdue support uses the configured response target. These are account
and record counts, rather than estimates of real-time activity.

Member management supports account suspension/reactivation and manual
verification review. Verification changes the review status; it does not claim
that an external NIN service verified the identity. Marketplace records include
products, current categories, account/listing reports, and flagged market price
reports. Operators can activate/inactivate listings with stock and record
moderation findings. Categories are derived from the existing listing taxonomy.

Orders and deliveries show platform-wide order statuses, cancellations, delivery
partners, and the latest recorded location. Complaints, disputes, and refund
requests are support tickets linked to a member and optionally to that member's
buyer/seller order. Staff can assign enabled staff, record progress and
resolution notes, and inspect the permanent resolution history.

Finance shows wallet balances, deposits, withdrawals, transfers, commissions,
payment/refund statuses, payments requiring investigation, and wallet ledger
settlement records. Transfer approval and withdrawal rejection/approval/status
checks use the existing wallet service. Bank payouts require the configured
payment provider. Resolving a refund ticket records the case outcome; provider
refund execution continues through the existing payment operations process.

Analytics show monthly member/farmer growth, sales and refunds by currency,
marketplace category activity, support workload and resolution timing, and
delivery status totals. Searches are parameterized, and tables show up to 25
records per page.

## Settings and audit records

Super Admin can edit only the approved settings: support default priority,
response target, dispute and listing policies, and ticket opened/resolved
notification templates. Defaults and response targets affect ticket workflows;
policies appear in the relevant workspace. Ticket creation/resolution creates
in-app notifications using the saved templates.

Management changes require a reason and a session-bound form token. Changes and
their audit record commit together. Logs include actor name/role, action,
target, safe before/after values, timestamp, and success/failure. Passwords and
bank account numbers are excluded from audit values. Audit logs and ticket
history reject updates and deletes.

Payment provider actions record an intent before calling the wallet service and
an observed result afterward, since the service owns its financial transactions.
An intent without a completion record indicates an interrupted operation to
investigate. Existing member-facing actions outside this workspace retain their
existing domain histories.

## Validation

```sh
go test ./internal/models ./internal/handlers ./internal/routes
TEST_DATABASE_URL=postgres://test_role@localhost/test_server go test ./internal/integration -run TestAdminWorkspacePermissionsAndWorkflows -v
```

The database test creates and removes its own database. It checks all ten
sections with saved records, role restrictions and CSRF rejection, staff account
creation, disabled/suspended logins, ticket history and notifications, listing
moderation, verification, wallet decisions, pagination, escaping, and audit
protection. The test role needs `CREATEDB`; use a disposable server.
