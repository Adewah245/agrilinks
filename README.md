# Agro-Shield — Structured Project Documentation

**Document status:** Living architecture document  
**Project:** Agro-Shield  
**Primary backend:** Go  
**Frontend:** HTML, CSS, JavaScript  
**Database:** PostgreSQL  
**Payment provider:** B-Moni (`bmoni`)  
**Primary deployment database direction:** Neon PostgreSQL  
**Last updated:** 2026-09-08

---

## 1. Project Overview

Agro-Shield is an agricultural technology platform designed to connect people with agriculture through three major user intents:

1. **I want to buy**
2. **I want to sell**
3. **I want to learn**

The platform is being structured so that it is not limited to crop farming. It should support different agricultural activities, including crop farming, livestock, poultry, fisheries, and other farming categories as the system grows.

The architecture is being reorganized from the earlier crop-centered implementation into a more general agricultural platform.

---

## 2. Core Product Direction

### 2.1 Buy

A user can discover agricultural products through the marketplace.

Expected capabilities include:

- Browse products
- View product details
- View seller/farmer information
- Add products to a cart
- Negotiate where supported
- Place orders
- Communicate with sellers
- Receive notifications
- Complete transactions through the supported payment infrastructure
- Review products after a transaction

### 2.2 Sell

A user who sells agricultural products should be able to:

- Create a seller/farmer profile
- Associate themselves with relevant farming types
- Create product listings
- Specify quantity and unit
- Set prices
- Provide product descriptions
- Provide location information
- Receive buyer conversations
- Negotiate with buyers
- Manage orders
- Receive transaction information
- Receive notifications
- Build reputation through product reviews

### 2.3 Learn

Learning is a first-class product direction, not simply an extra page.

A learner should be able to:

1. Choose a farming type they want to learn.
2. Enter an agricultural learning space.
3. Create or join a learning room/community.
4. Learn the selected agricultural activity.
5. Interact with an AI learning assistant.
6. Progress toward becoming that type of farmer.

The learner dashboard is intentionally different from the normal farmer/seller dashboard.

The AI learning experience should guide the learner continuously, including topics such as:

- How to start
- Required conditions
- Temperature ranges
- Farming environment
- Feeding/care where applicable
- Growth or maturity period
- Common problems
- Best practices
- Product/farm readiness
- Next steps

---

# 3. Architectural Principles

The new Agro-Shield structure follows these principles:

### 3.1 Generalize agriculture

Avoid designing the core platform around `crop` when the concept is actually an agricultural product, farming activity, farmer, learner, or organization.

### 3.2 Separate concerns

The system should keep:

- Users
- Farming types
- Locations
- Organizations
- Products
- Orders
- Negotiations
- Communication
- Notifications
- Transactions
- Reviews
- Learning

as distinct concepts.

### 3.3 Database-first clarity

Every migration should represent a clear domain concept.

### 3.4 Avoid technical debt deliberately

If a requirement is already clearly known and the database/backend needs to support it, the structure should support it now rather than knowingly creating a design that will require another migration later.

### 3.5 Architecture-first migration workflow

Before changing a migration/schema beyond straightforward extraction:

1. Explain the current/old structure.
2. Explain the proposed structure.
3. Explain why the change is needed.
4. Discuss reasonable alternatives and trade-offs.
5. Agree on the design.
6. Implement the migration.
7. Verify the migration and dependent code.

This workflow applies to future Agro-Shield database/schema work.

---

# 4. High-Level System Architecture

```text
                         ┌─────────────────────┐
                         │      Frontend       │
                         │ HTML / CSS / JS     │
                         └──────────┬──────────┘
                                    │
                                    │ HTTP
                                    ▼
                         ┌─────────────────────┐
                         │    Go Backend       │
                         │   net/http based    │
                         └──────────┬──────────┘
                                    │
          ┌─────────────────────────┼──────────────────────────┐
          │                         │                          │
          ▼                         ▼                          ▼
 ┌─────────────────┐      ┌─────────────────┐       ┌─────────────────┐
 │ Domain Services │      │ Repositories    │       │ External        │
 │ / Business      │      │ / Data Access   │       │ Integrations    │
 │ Logic           │      │                 │       │                 │
 └────────┬────────┘      └────────┬────────┘       └───────┬─────────┘
          │                        │                          │
          └────────────────────────┼──────────────────────────┘
                                   ▼
                         ┌─────────────────────┐
                         │    PostgreSQL       │
                         │      / Neon         │
                         └─────────────────────┘
                                   │
                 ┌─────────────────┼──────────────────┐
                 │                 │                  │
                 ▼                 ▼                  ▼
             Payments          Weather            Future
             B-Moni            Integration        Services
```

---

# 5. Technology Stack

## Backend

- Go
- Standard `net/http` architecture
- PostgreSQL database
- `pgx/v5` PostgreSQL driver/library
- Service/repository separation

The project should not introduce a web framework such as Gin unless the architecture is deliberately reconsidered.

## Frontend

Current direction:

- HTML
- CSS
- JavaScript

The frontend is not being rebuilt around React at this stage.

## Database

- PostgreSQL
- Neon is the cloud PostgreSQL direction
- SQL migrations are maintained as separate migration files

---

# 6. Proposed Backend Organization

The backend should follow domain-oriented separation rather than placing all logic into large handlers.

A representative structure is:

```text
backend/
├── cmd/
│   └── ...
├── internal/
│   ├── handlers/
│   ├── services/
│   ├── repositories/
│   ├── models/
│   ├── middleware/
│   └── ...
├── migrations/
│   ├── 001_users.sql
│   ├── 002_farming_types.sql
│   ├── ...
│   └── ...
├── go.mod
└── ...
```

The exact directory names may evolve as the implementation is cleaned up, but the responsibility boundaries should remain clear.

---

# 7. User Domain

## 7.1 Users

The `users` table is the central identity record.

The new role model includes:

- `farmer`
- `buyer`
- `learner`

The role constraint must therefore support the learner role.

Conceptually:

```text
User
 ├── role
 ├── identity/contact information
 ├── verification
 ├── location
 └── farming interests/types
```

A user should not be forced into a crop-only identity.

---

# 8. Farming Types

The platform introduces a reusable `farming_types` concept.

Examples of possible farming types include:

- Crop farming
- Livestock
- Poultry
- Fisheries
- Other agricultural activities

The actual seed data can expand as the project develops.

The purpose of this table is to provide one central definition of agricultural activities that can be reused by:

- Learners
- Farmers
- Profiles
- Learning rooms
- Educational content
- Recommendations
- Future analytics

---

# 9. User Farming Types

A many-to-many relationship is required between users and farming types.

Conceptually:

```text
users
  │
  │ many-to-many
  ▼
user_farming_types
  ▲
  │
  │
farming_types
```

This allows one user to have multiple farming interests or specializations.

For example:

```text
User A
 ├── Poultry
 └── Crop Farming

User B
 ├── Fisheries
 └── Livestock
```

This is preferable to putting a single `farming_type` string directly on the user.

---

# 10. User Verification

The architecture includes a `user_verifications` concept.

Verification exists to improve trust and reduce fraud.

Possible verification categories can evolve independently from the main user table.

Sensitive verification information should not be unnecessarily exposed publicly.

NIN-related verification was discussed as a possible trust mechanism, but verification data should remain private and should not become a public marketplace profile field.

---

# 11. Nigerian Geographic Structure

Agro-Shield is designed around Nigeria and therefore requires a reusable geographic hierarchy.

The planned structure is:

```text
Nigeria
  │
  ├── States / FCT
  │      │
  │      └── LGAs
  │              │
  │              └── Communities
```

Core geographic concepts:

- `states`
- `lgas`
- `communities`

The database has been planned to contain the Nigerian states/FCT and the 774 LGAs as seed data.

Communities provide a finer location level where needed.

This structure should be reused instead of storing every location as unstructured text.

---

# 12. Organizations

Agro-Shield supports agricultural organizations.

Core concepts:

- `organizations`
- `organization_members`

An organization may represent:

- Farmer associations
- Cooperatives
- Agricultural groups
- Other relevant agricultural organizations

Membership should be modeled separately so that a user can belong to an organization without duplicating user information.

Conceptually:

```text
users
  │
  │
  ▼
organization_members
  ▲
  │
  │
organizations
```

Organization membership can later support:

- Roles
- Permissions
- Announcements
- Group communication
- Learning communities
- Agricultural collaboration

---

# 13. Announcements

An `announcements` concept is planned for organization/platform communication.

Announcements can be associated with the appropriate organization or platform context.

This keeps communication separate from direct chat messages.

---

# 14. Marketplace

The marketplace is being generalized from the old crop-specific model.

The new core concept is:

```text
products
```

rather than treating every marketplace item as a `crop`.

A product should support concepts such as:

- Owner
- Owner type
- Name
- Description
- Category
- Quantity
- Unit
- Price
- Location
- Status
- Creation/update timestamps

The product ownership model uses:

```text
owner_type
owner_id
```

This gives the marketplace room to support different ownership models rather than assuming every product is owned by one specific legacy entity.

---

# 15. Product Lifecycle

A simplified product lifecycle is:

```text
Create
  ↓
Active
  ↓
Viewed
  ↓
Negotiated (optional)
  ↓
Ordered
  ↓
Transaction
  ↓
Reviewed
```

Product status should be controlled by the marketplace domain rather than being scattered across unrelated services.

---

# 16. Cart

The marketplace supports a cart concept.

The cart connects:

```text
User
  ↓
Cart Items
  ↓
Products
```

Cart items should reference the generalized product rather than a legacy crop-only entity.

---

# 17. Orders

Orders represent the buyer's commitment to purchase products.

The order domain should remain separate from negotiation.

A simplified flow:

```text
Product
   ↓
Cart
   ↓
Order
   ↓
Payment / Transaction
   ↓
Fulfillment
```

Orders should preserve enough information to determine:

- Buyer
- Product/order items
- Quantity
- Agreed price
- Order state
- Relevant timestamps
- Delivery information where applicable

---

# 18. Delivery Partners

The architecture includes a `delivery_partners` concept.

This allows logistics to be separated from the product and order entities.

Future delivery capabilities can include:

- Delivery partner registration
- Assignment
- Delivery status
- Tracking
- Delivery fees

---

# 19. Negotiations

Negotiation is being generalized to work with the new product model.

The important architectural change is:

```text
Old:
Negotiation → crop_id

New:
Negotiation → product_id
```

The reason is that marketplace negotiations should happen around marketplace products, not a legacy crop-specific entity.

Negotiations can contain:

- Buyer
- Seller
- Product
- Negotiation status
- Current agreed/proposed price
- Expiration information
- Round information
- Timestamps

The current business rule discussed for the negotiation system is:

- Maximum of 5 rounds
- Negotiation expires after 24 hours

---

# 20. Negotiation Messages

Negotiation messages belong to the negotiation domain.

Conceptually:

```text
Negotiation
   │
   └── Negotiation Messages
```

This keeps negotiation-specific price discussions separate from ordinary chat.

---

# 21. Communication System

Agro-Shield has a broader communication domain independent of negotiations.

Core concepts include:

- `conversations`
- `conversation_members`
- `chat_messages`

Conceptually:

```text
Conversation
   │
   ├── Members
   │
   └── Chat Messages
```

This supports direct communication between users and can later support broader group communication.

Negotiation messages should not be confused with general chat messages.

---

# 22. Notifications

The notification system provides user-facing alerts for important events.

Possible events include:

- New message
- New negotiation activity
- Negotiation accepted/rejected
- Order changes
- Payment/transaction events
- Organization announcements
- Learning events
- Other platform events

Notifications should remain a separate domain so individual services do not need to implement notification persistence themselves.

---

# 23. Transactions and Payments

Agro-Shield separates transactions from orders.

An order represents the commercial order.

A transaction represents the financial movement/state.

The selected payment provider direction is:

**B-Moni (`bmoni`)**

The architecture should therefore allow payment integration through a dedicated payment/transaction boundary rather than coupling payment-provider logic directly to marketplace handlers.

Conceptually:

```text
Order
  ↓
Payment Request
  ↓
B-Moni
  ↓
Transaction
  ↓
Payment Result
  ↓
Order/Notification Update
```

Provider-specific implementation details should remain isolated so that the core business logic does not become dependent on provider-specific code.

---

# 24. Product Reviews

The platform includes `product_reviews`.

Reviews should be tied to completed/relevant transactions or orders so that the system can avoid arbitrary reviews detached from actual marketplace activity.

The review system supports:

- Buyer feedback
- Product reputation
- Seller/product trust
- Future marketplace ranking/recommendation features

---

# 25. Weather Integration

Weather is considered an important cross-platform capability.

The current direction is that weather information does **not** need to become a primary database table merely to display current/future weather.

Instead, weather should be treated as an integration/service.

Potential consumers include:

- Farmer dashboard
- Learner dashboard
- Farming guidance
- Crop-related advice
- Storage monitoring
- Agricultural recommendations

The architecture should allow weather conditions to be connected to storage and agricultural operations in the future without forcing the weather provider's raw response into the core database model.

---

# 26. Learning Architecture

Learning is a first-class domain.

A simplified conceptual structure is:

```text
User
  │
  ▼
Select Farming Type
  │
  ▼
Learning Space / Room
  │
  ├── Learning Content
  ├── Community / Members
  ├── AI Learning Assistant
  └── Progress
```

The learner experience should be designed independently from the farmer dashboard.

The frontend is expected to lead this feature first.

Database/backend support should follow after the learning experience and data requirements are sufficiently defined.

---

# 27. AI Assistant

Agro-Shield's AI capability is not limited to a generic chatbot.

For learning, the AI assistant should act as an agricultural learning guide.

The experience should be conversational and continuous.

Example flow:

```text
Learner:
"I want to learn poultry farming."

AI:
"Great. Let's begin by understanding the type of poultry
you want to raise..."

        ↓

Environment
        ↓
Housing
        ↓
Temperature
        ↓
Feeding
        ↓
Health
        ↓
Growth
        ↓
Maturity
        ↓
Next practical step
```

The AI system should eventually be able to use the learner's selected farming type and progress context.

---

# 28. Dashboard Model

Agro-Shield should not have one giant dashboard attempting to serve every type of user.

The dashboard should be intent/domain aware.

### Buyer-oriented dashboard

Focus:

- Marketplace
- Cart
- Orders
- Negotiations
- Messages
- Notifications

### Farmer/seller-oriented dashboard

Focus:

- Products
- Sales
- Orders
- Negotiations
- Messages
- Agricultural information
- Weather
- Storage where applicable

### Learner-oriented dashboard

Focus:

- Learning goals
- Selected farming type
- Learning room
- AI learning assistant
- Progress
- Educational resources
- Weather/agricultural guidance where useful

---

# 29. Existing/Legacy Architecture vs New Architecture

The previous implementation was strongly centered around entities such as:

```text
Farmers
Crops
Diagnoses
Crop listings
Crop negotiations
```

The new structure moves toward:

```text
Users
Farming Types
Locations
Organizations
Products
Orders
Negotiations
Conversations
Notifications
Transactions
Reviews
Learning
```

The goal is not to throw away useful existing business logic.

Instead, useful logic should be migrated into the new domain structure carefully.

---

# 30. Diagnosis / Agricultural Intelligence

The previous system included diagnosis functionality and diagnosis media.

These concepts can remain useful, but they should not dictate the entire platform architecture.

Diagnosis should eventually be treated as an agricultural intelligence feature that can work with different farming types where appropriate.

The database design should avoid assuming that every diagnosis is necessarily about a crop.

---

# 31. Data Relationship Overview

A simplified relationship model:

```text
                         ┌───────────────┐
                         │     Users     │
                         └───────┬───────┘
                                 │
              ┌──────────────────┼──────────────────┐
              │                  │                  │
              ▼                  ▼                  ▼
      User Farming Types   Verifications       Organizations
              │                                     │
              ▼                                     ▼
       Farming Types                         Organization Members
              │
              │
              ▼
        Learning Spaces
              │
              ▼
       Learning Progress


Users ──────────────── Products
  │                       │
  │                       │
  ├── Cart Items ─────────┤
  │                       │
  ├── Orders ─────────────┤
  │                       │
  ├── Negotiations ───────┤
  │                       │
  └── Conversations       │
                          │
                          ▼
                    Product Reviews

Orders
  │
  ▼
Transactions
  │
  ▼
B-Moni

Users
  │
  ▼
Notifications
```

---

# 32. Migration Strategy

The migration structure is being rebuilt carefully.

The early migration direction includes concepts such as:

```text
001_users.sql
002_farming_types.sql
003_user_farming_types.sql
004_user_verifications.sql
005_states.sql
006_lgas.sql
007_communities.sql
...
```

The exact numbering may change as migrations are finalized.

The important rule is dependency order.

For example:

```text
Users
  ↓
User-dependent tables

States
  ↓
LGAs
  ↓
Communities

Farming Types
  ↓
User Farming Types

Users
  ↓
Organizations / Memberships

Products
  ↓
Negotiations
  ↓
Orders / related marketplace operations
```

Foreign-key dependencies must be considered before changing migration order.

---

# 33. Migration Safety Rules

Before modifying an existing migration:

### Step 1 — Inspect

Determine:

- What the table currently represents.
- Which columns already exist.
- Which code depends on it.
- Which foreign keys depend on it.
- Whether data already exists.

### Step 2 — Propose

Explain:

- Current design
- New design
- Why the new design is better
- Alternatives
- Trade-offs

### Step 3 — Agree

Do not silently introduce major architectural decisions.

### Step 4 — Implement

Make the smallest safe change that produces the agreed architecture.

### Step 5 — Verify

Run:

- Migration
- Application compilation
- Tests
- Relevant database checks
- Relevant API/handler checks

---

# 34. Legacy Migration Principle

Legacy compatibility migrations should not become permanent architecture.

If an old column such as:

```text
crop_id
image_name
category
```

is no longer part of the new agreed domain model, the team should determine whether:

1. It should be migrated to the new model.
2. It should temporarily remain for compatibility.
3. It should be removed.

A compatibility layer should have a clear reason and should not silently recreate the old architecture.

---

# 35. Security and Trust

Agro-Shield should prioritize trust because the platform involves:

- Farmers
- Buyers
- Sellers
- Money
- Negotiation
- Agricultural organizations

Important principles include:

- Passwords must be securely hashed.
- Sensitive verification information must remain private.
- Authentication and authorization must be separated.
- Users should only access resources they are authorized to access.
- Payment information should be handled through the payment provider/integration boundary.
- Public profiles should expose only appropriate information.
- Database constraints should protect important relationships.

---

# 36. Current User Capability Direction

A basic user concept was previously defined with capabilities such as:

### Available

- Browse marketplace
- View product details
- View farmer/seller profile
- Chat
- Use AI assistant
- Add to wishlist

### Restricted / not automatically available

- Buy
- Sell
- Negotiate
- Place orders
- Payment/USSD transactions

These capabilities should eventually be represented through proper authorization and user state rather than relying only on frontend hiding.

---

# 37. USSD Direction

USSD was discussed as a future/important access channel for registered users.

The architecture should keep USSD as an interface into the same core business services rather than implementing a separate business system.

Conceptually:

```text
Web Frontend ─────┐
                  │
Mobile/Other UI ──┼──> Go Business Services ──> PostgreSQL
                  │
USSD ─────────────┘
```

This avoids duplicating business logic.

---

# 38. API Design Principle

Handlers should primarily handle HTTP concerns:

- Request parsing
- Authentication context
- Validation at the HTTP boundary
- Calling services
- HTTP response formatting

Business rules should live in services.

Database operations should live in repositories/data-access layers.

Conceptually:

```text
HTTP Request
     ↓
Handler
     ↓
Service
     ↓
Repository
     ↓
PostgreSQL
```

Not:

```text
HTTP Handler
   ↓
Huge block of SQL + business logic + payment logic + notifications
```

---

# 39. Service Boundaries

Likely service boundaries include:

```text
UserService
FarmingTypeService
OrganizationService
ProductService
CartService
OrderService
NegotiationService
ConversationService
NotificationService
Transaction/PaymentService
ReviewService
LearningService
WeatherService
AI Assistant Service
```

Not every service needs to be created immediately.

A service should be introduced when it represents a real domain responsibility.

---

# 40. Repository Boundaries

Repositories should focus on persistence.

Examples:

```text
UserRepository
FarmingTypeRepository
OrganizationRepository
ProductRepository
CartRepository
OrderRepository
NegotiationRepository
NegotiationMessageRepository
ConversationRepository
ConversationMemberRepository
ChatMessageRepository
NotificationRepository
TransactionRepository
ProductReviewRepository
```

Repositories should not contain high-level business workflows.

---

# 41. Frontend Direction

The frontend should reflect the same domain structure.

Suggested major areas:

```text
frontend/
├── assets/
├── pages/
├── components/
├── js/
└── ...
```

The exact directory layout can evolve, but pages should be organized around user intent and domain features.

Important frontend areas include:

- Landing page
- Authentication
- Buyer/marketplace experience
- Seller/farmer experience
- Learning experience
- AI assistant
- Product details
- Cart
- Orders
- Negotiations
- Chat
- Notifications
- Organization pages
- Profile/settings

---

# 42. Development Priorities

The project should be developed in a controlled order.

## Phase 1 — Foundation

- Users
- Authentication
- Roles
- Farming types
- User farming types
- Verification
- Nigeria locations

## Phase 2 — Marketplace

- Products
- Product categories
- Cart
- Orders
- Seller/buyer workflows

## Phase 3 — Communication

- Conversations
- Conversation members
- Chat messages
- Notifications

## Phase 4 — Negotiation

- Product-based negotiations
- Negotiation messages
- Five-round rule
- 24-hour expiration

## Phase 5 — Payments

- Transactions
- B-Moni integration
- Payment state handling

## Phase 6 — Logistics

- Delivery partners
- Delivery workflow
- Delivery tracking

## Phase 7 — Trust

- Product reviews
- Verification improvements
- Reputation mechanisms

## Phase 8 — Learning

- Learning dashboard
- Farming type selection
- Learning rooms/spaces
- AI learning assistant
- Progress tracking

## Phase 9 — Agricultural intelligence

- Weather
- Diagnosis
- Storage/weather connection
- Future agricultural recommendations

---

# 43. Important Architectural Decisions

| Area | Direction |
|---|---|
| Backend language | Go |
| HTTP approach | `net/http` |
| Frontend | HTML/CSS/JavaScript |
| Database | PostgreSQL |
| Cloud PostgreSQL | Neon direction |
| Payment provider | B-Moni |
| Core identity | Users |
| User roles | Farmer, Buyer, Learner |
| Farming model | Farming Types |
| Marketplace entity | Products |
| Negotiation target | Product |
| Communication | Conversations + Chat Messages |
| Negotiation communication | Negotiation Messages |
| Notifications | Separate notification domain |
| Financial model | Transactions |
| Reviews | Product Reviews |
| Nigerian geography | States → LGAs → Communities |
| Organizations | Organizations + Organization Members |
| Learning | First-class domain |
| Weather | Integration/service, not automatically a core table |
| Learning frontend | Frontend-first |
| Schema workflow | Architecture-first + collaborative |

---

# 44. What We Should Avoid

### Avoid crop-only architecture

Do not introduce new core features using `crop` when `product`, `farming_type`, or another broader domain concept is the correct abstraction.

### Avoid giant tables

Do not put unrelated concerns into `users`, `products`, or `orders`.

### Avoid business logic in handlers

Handlers should not become the application's business layer.

### Avoid provider lock-in

B-Moni-specific logic should remain behind a payment boundary.

### Avoid duplicate location strings

Use the structured Nigerian geographic model where appropriate.

### Avoid hidden architectural changes

Major schema changes must be explained and agreed upon before implementation.

### Avoid premature complexity

Not every future feature needs to be implemented immediately. The architecture should provide clean extension points without building unnecessary systems.

---

# 45. Definition of Done for a Schema Change

A schema change is not considered complete merely because the SQL executes.

A completed schema change should have:

- Agreed domain meaning
- Correct dependency order
- Correct foreign keys
- Appropriate constraints
- Appropriate indexes where needed
- Compatible Go models
- Compatible repositories
- Compatible services
- Compatible handlers
- Tests or verification
- Successful migration
- No unexplained legacy errors

---

# 46. Project Mental Model

The most important mental model for the new Agro-Shield architecture is:

```text
                    AGRICULTURE
                         │
       ┌─────────────────┼─────────────────┐
       │                 │                 │
      BUY               SELL              LEARN
       │                 │                 │
       ▼                 ▼                 ▼
   Products          Products        Farming Types
       │                 │                 │
       ├───────┐         │                 ▼
       │       │         │          Learning Spaces
      Cart  Orders       │                 │
               │         │                 ▼
               ▼         ▼           AI Learning
            Transactions │
               │         │
             B-Moni      │
                         │
                   Negotiations
                         │
                         ▼
                  Conversations
                         │
                         ▼
                   Notifications
```

Around all of these are the shared platform foundations:

```text
Users
Roles
Verification
Locations
Organizations
Farming Types
Weather
Security
```

---

# 47. Current Architectural Goal

The goal is to make Agro-Shield a **general agricultural platform**, not merely a crop marketplace.

The new architecture should allow the platform to grow from:

```text
Crop marketplace
```

into:

```text
Agricultural ecosystem
```

where a person can:

```text
Discover agriculture
       ↓
Learn agriculture
       ↓
Become a farmer
       ↓
Produce agricultural goods
       ↓
Sell products
       ↓
Negotiate
       ↓
Receive payment
       ↓
Deliver
       ↓
Build reputation
       ↓
Join agricultural organizations
       ↓
Continue learning and improving
```

This is the long-term architectural direction.

---

# 48. Documentation Status

This document is a **living document**.

It should be updated whenever a major architectural decision is agreed upon.

When implementation differs from this document, the discrepancy should be identified and resolved rather than allowing the documentation and codebase to silently diverge.

The database schema remains subject to the architecture-first review process before major migrations are changed.

---

## End of Agro-Shield Structured Architecture Documentation