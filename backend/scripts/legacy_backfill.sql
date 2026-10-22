-- Backfill current Agro-Shield tables from the archived crop/farmer schema.
-- The script preserves existing ids where other legacy records reference them
-- and leaves the original data untouched in legacy_agro_shield.

-- ---------------------------------------------------------------------------
-- People and bank details
-- ---------------------------------------------------------------------------

INSERT INTO users (
    id,
    full_name,
    phone,
    email,
    password_hash,
    role,
    account_type,
    location,
    lga,
    state,
    country,
    organisation_name,
    photo_url,
    is_admin,
    deleted_at,
    created_at,
    updated_at
)
SELECT
    f.id,
    COALESCE(NULLIF(BTRIM(f.full_name), ''), 'Agro-Shield user'),
    f.phone,
    CASE
        WHEN NULLIF(BTRIM(f.email), '') IS NULL THEN NULL
        WHEN ROW_NUMBER() OVER (
            PARTITION BY LOWER(BTRIM(f.email))
            ORDER BY f.id
        ) = 1 THEN BTRIM(f.email)
        ELSE NULL
    END,
    f.password_hash,
    CASE
        WHEN f.role IN ('farmer', 'buyer', 'ministry', 'organisation', 'learner') THEN f.role
        ELSE 'farmer'
    END,
    'individual',
    f.location,
    f.lga,
    f.state,
    COALESCE(NULLIF(BTRIM(f.country), ''), 'Nigeria'),
    NULL,
    NULLIF(BTRIM(f.photo_url), ''),
    COALESCE(f.is_admin, FALSE),
    f.deleted_at,
    f.created_at,
    COALESCE(f.updated_at, f.created_at)
FROM legacy_agro_shield.farmers f
WHERE BTRIM(f.phone) <> ''
  AND BTRIM(f.password_hash) <> ''
ON CONFLICT (id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('users', 'id'),
    COALESCE((SELECT MAX(id) FROM users), 0) + 1,
    FALSE
);

INSERT INTO user_bank_accounts (
    user_id,
    bank_name,
    bank_code,
    account_name,
    account_number,
    created_at,
    updated_at
)
SELECT
    f.id,
    f.bank_name,
    f.bank_code,
    f.account_name,
    f.account_number,
    f.created_at,
    COALESCE(f.updated_at, f.created_at)
FROM legacy_agro_shield.farmers f
JOIN users u ON u.id = f.id
WHERE BTRIM(f.bank_name) <> ''
  AND BTRIM(f.bank_code) <> ''
  AND BTRIM(f.account_name) <> ''
  AND BTRIM(f.account_number) <> ''
ON CONFLICT (user_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Products, images, prices, and stock history
-- ---------------------------------------------------------------------------

INSERT INTO products (
    id,
    owner_type,
    owner_id,
    name,
    description,
    category,
    quantity,
    unit,
    price,
    state_id,
    lga_id,
    latitude,
    longitude,
    location_accuracy,
    status,
    initial_listed_quantity,
    first_listed_at,
    first_order_at,
    sold_out_at,
    created_at,
    updated_at
)
SELECT
    c.id,
    'user',
    c.farmer_id,
    COALESCE(NULLIF(BTRIM(c.name), ''), 'Agricultural product'),
    NULL,
    'crops',
    GREATEST(COALESCE(c.quantity, 0), 0),
    COALESCE(NULLIF(BTRIM(c.unit), ''), 'unit'),
    GREATEST(COALESCE(c.price_per_unit, 0), 0),
    s.id,
    l.id,
    c.latitude,
    c.longitude,
    c.location_accuracy,
    CASE
        WHEN c.listed_for_sale IS TRUE AND COALESCE(c.quantity, 0) > 0 THEN 'active'
        WHEN c.listed_for_sale IS TRUE THEN 'sold_out'
        ELSE 'inactive'
    END,
    GREATEST(COALESCE(c.initial_listed_quantity, c.quantity, 0), 0),
    c.first_listed_at,
    c.first_order_at,
    c.sold_out_at,
    c.created_at,
    COALESCE(c.updated_at, c.created_at)
FROM legacy_agro_shield.crops c
JOIN users u ON u.id = c.farmer_id
LEFT JOIN states s ON LOWER(BTRIM(s.name)) = LOWER(BTRIM(c.state))
LEFT JOIN lgas l ON l.state_id = s.id AND LOWER(BTRIM(l.name)) = LOWER(BTRIM(c.lga))
ON CONFLICT (id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('products', 'id'),
    COALESCE((SELECT MAX(id) FROM products), 0) + 1,
    FALSE
);

INSERT INTO product_images (product_id, image_url, is_primary, sort_order, created_at)
SELECT
    ci.crop_id,
    ci.image_url,
    COALESCE(ci.is_primary, FALSE),
    GREATEST(COALESCE(ci.sort_order, 0), 0),
    ci.created_at
FROM legacy_agro_shield.crop_images ci
JOIN products p ON p.id = ci.crop_id
WHERE BTRIM(ci.image_url) <> ''
  AND NOT EXISTS (
      SELECT 1
      FROM product_images pi
      WHERE pi.product_id = ci.crop_id
        AND pi.image_url = ci.image_url
  );

INSERT INTO product_price_history (product_id, price, unit, source, recorded_at)
SELECT
    ph.crop_id,
    GREATEST(COALESCE(ph.price_per_unit, 0), 0),
    COALESCE(NULLIF(BTRIM(ph.unit), ''), 'unit'),
    COALESCE(NULLIF(BTRIM(ph.source), ''), 'Legacy Agro-Shield'),
    ph.observed_at
FROM legacy_agro_shield.produce_price_history ph
JOIN products p ON p.id = ph.crop_id
WHERE NOT EXISTS (
    SELECT 1
    FROM product_price_history current_ph
    WHERE current_ph.product_id = ph.crop_id
      AND current_ph.recorded_at = ph.observed_at
      AND current_ph.price = ph.price_per_unit
);

INSERT INTO inventory_history (
    product_id,
    quantity_before,
    quantity_change,
    quantity_after,
    reason,
    created_at
)
SELECT
    ih.crop_id,
    GREATEST(COALESCE(ih.quantity_before, 0), 0),
    COALESCE(ih.quantity_change, 0),
    GREATEST(COALESCE(ih.quantity_after, 0), 0),
    COALESCE(NULLIF(BTRIM(ih.reason), ''), 'Legacy inventory update'),
    ih.created_at
FROM legacy_agro_shield.inventory_history ih
JOIN products p ON p.id = ih.crop_id
WHERE NOT EXISTS (
    SELECT 1
    FROM inventory_history current_ih
    WHERE current_ih.product_id = ih.crop_id
      AND current_ih.created_at = ih.created_at
      AND current_ih.reason = ih.reason
);

INSERT INTO product_status_history (
    product_id,
    old_status,
    new_status,
    reason,
    changed_by_type,
    changed_by_id,
    created_at
)
SELECT
    p.id,
    NULL,
    p.status,
    'Imported from the legacy listing',
    'system',
    NULL,
    p.updated_at
FROM products p
WHERE NOT EXISTS (
    SELECT 1
    FROM product_status_history psh
    WHERE psh.product_id = p.id
      AND psh.reason = 'Imported from the legacy listing'
);

-- ---------------------------------------------------------------------------
-- Orders
-- ---------------------------------------------------------------------------

INSERT INTO orders (id, buyer_id, status, total_amount, created_at, updated_at)
SELECT
    o.id,
    o.buyer_id,
    CASE
        WHEN o.status IN ('pending', 'confirmed', 'processing', 'ready_for_delivery', 'shipped', 'delivered', 'cancelled') THEN o.status
        ELSE 'pending'
    END,
    GREATEST(COALESCE(o.total_price, 0), 0),
    o.created_at,
    o.created_at
FROM legacy_agro_shield.orders o
JOIN users u ON u.id = o.buyer_id
ON CONFLICT (id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('orders', 'id'),
    COALESCE((SELECT MAX(id) FROM orders), 0) + 1,
    FALSE
);

INSERT INTO order_items (order_id, product_id, quantity, unit_price, total_price, created_at)
SELECT
    o.id,
    o.crop_id,
    o.quantity,
    GREATEST(COALESCE(o.total_price, 0), 0) / o.quantity,
    GREATEST(COALESCE(o.total_price, 0), 0),
    o.created_at
FROM legacy_agro_shield.orders o
JOIN orders current_order ON current_order.id = o.id
JOIN products p ON p.id = o.crop_id
WHERE o.quantity > 0
  AND NOT EXISTS (
      SELECT 1
      FROM order_items oi
      WHERE oi.order_id = o.id
        AND oi.product_id = o.crop_id
  );

-- ---------------------------------------------------------------------------
-- Negotiations and messages
-- ---------------------------------------------------------------------------

INSERT INTO negotiations (
    id,
    product_id,
    buyer_id,
    quantity,
    initial_offer_price,
    current_offer_price,
    round_count,
    max_rounds,
    status,
    expires_at,
    accepted_at,
    closed_at,
    created_at,
    updated_at
)
SELECT
    n.id,
    n.crop_id,
    n.buyer_id,
    GREATEST(COALESCE(n.quantity, 1), 0.000001),
    n.initial_offer_price,
    n.current_offer_price,
    LEAST(GREATEST(COALESCE(n.round_count, 0), 0), COALESCE(NULLIF(n.max_rounds, 0), 5)),
    COALESCE(NULLIF(n.max_rounds, 0), 5),
    CASE
        WHEN n.status IN ('open', 'accepted', 'rejected', 'expired', 'closed') THEN n.status
        ELSE 'open'
    END,
    GREATEST(
        COALESCE(n.expires_at, n.created_at + INTERVAL '24 hours'),
        n.created_at + INTERVAL '1 second'
    ),
    CASE WHEN n.status = 'accepted' THEN COALESCE(n.accepted_at,n.created_at) END,
    CASE WHEN n.status IN ('rejected', 'expired', 'closed') THEN COALESCE(n.closed_at,n.created_at) END,
    n.created_at,
    n.created_at
FROM legacy_agro_shield.negotiations n
JOIN products p ON p.id = n.crop_id
JOIN users u ON u.id = n.buyer_id
ON CONFLICT (id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('negotiations', 'id'),
    COALESCE((SELECT MAX(id) FROM negotiations), 0) + 1,
    FALSE
);

INSERT INTO negotiation_messages (
    id,
    negotiation_id,
    sender_id,
    message_type,
    message,
    offer_price,
    offer_status,
    created_at
)
SELECT
    m.id,    m.negotiation_id,
    m.sender_id,
    CASE
        WHEN LOWER(BTRIM(m.message_type)) = 'offer'
         AND m.offer_price IS NOT NULL THEN 'offer'
        ELSE 'message'
    END,
    CASE
        WHEN LOWER(BTRIM(m.message_type)) = 'offer' THEN NULLIF(BTRIM(m.message), '')
        ELSE COALESCE(NULLIF(BTRIM(m.message), ''), 'Message from previous chat')
    END,
    CASE
        WHEN LOWER(BTRIM(m.message_type)) = 'offer' THEN GREATEST(m.offer_price, 0)
        ELSE NULL
    END,
    CASE
        WHEN LOWER(BTRIM(m.message_type)) = 'offer' THEN
            CASE
                WHEN m.offer_status IN ('pending', 'accepted', 'rejected') THEN m.offer_status
                ELSE 'pending'
            END
        ELSE 'none'
    END,
    m.created_at
FROM legacy_agro_shield.negotiation_messages m
JOIN negotiations n ON n.id = m.negotiation_id
JOIN users u ON u.id = m.sender_id
ON CONFLICT (id) DO NOTHING;

SELECT setval(pg_get_serial_sequence('negotiation_messages','id'),COALESCE((SELECT MAX(id) FROM negotiation_messages),0)+1,FALSE);

-- ---------------------------------------------------------------------------
-- Cart
-- ---------------------------------------------------------------------------

INSERT INTO cart_items (
    id,
    buyer_id,
    product_id,
    negotiation_id,
    quantity,
    price_per_unit,
    total_price,
    created_at,
    updated_at
)
SELECT
    ci.id,    ci.buyer_id,
    ci.crop_id,
    CASE WHEN EXISTS (SELECT 1 FROM negotiations n WHERE n.id = ci.negotiation_id) THEN ci.negotiation_id END,
    GREATEST(COALESCE(ci.quantity, 0.000001), 0.000001),
    GREATEST(COALESCE(ci.price_per_unit, 0), 0),
    GREATEST(COALESCE(ci.total_price, 0), 0),
    ci.created_at,
    ci.created_at
FROM legacy_agro_shield.cart_items ci
JOIN users u ON u.id = ci.buyer_id
JOIN products p ON p.id = ci.crop_id
ON CONFLICT (id) DO NOTHING;

SELECT setval(pg_get_serial_sequence('cart_items','id'),COALESCE((SELECT MAX(id) FROM cart_items),0)+1,FALSE);

-- ---------------------------------------------------------------------------
-- Conversations and messages
-- ---------------------------------------------------------------------------

INSERT INTO conversations (
    id,
    conversation_type,
    title,
    created_by,
    created_at,
    updated_at
)
SELECT
    c.id,
    CASE WHEN c.type = 'group' THEN 'group' ELSE 'private' END,
    NULLIF(BTRIM(c.name), ''),
    CASE WHEN EXISTS (SELECT 1 FROM users u WHERE u.id = c.created_by) THEN c.created_by END,
    c.created_at,
    COALESCE(c.updated_at, c.created_at)
FROM legacy_agro_shield.conversations c
ON CONFLICT (id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('conversations', 'id'),
    COALESCE((SELECT MAX(id) FROM conversations), 0) + 1,
    FALSE
);

INSERT INTO conversation_members (
    conversation_id,
    user_id,
    member_role,
    joined_at,
    last_read_at
)
SELECT
    cm.conversation_id,
    cm.user_id,
    CASE WHEN c.created_by=cm.user_id AND c.conversation_type='group' THEN 'admin' ELSE 'member' END,
    COALESCE(cm.joined_at, cm.last_read_at, CURRENT_TIMESTAMP),
    GREATEST(
        COALESCE(cm.last_read_at, cm.joined_at, CURRENT_TIMESTAMP),
        COALESCE(cm.joined_at, CURRENT_TIMESTAMP)
    )
FROM legacy_agro_shield.conversation_members cm
JOIN conversations c ON c.id = cm.conversation_id
JOIN users u ON u.id = cm.user_id
ON CONFLICT (conversation_id, user_id) DO NOTHING;

INSERT INTO chat_messages (
    id,
    conversation_id,
    sender_id,
    message_type,
    message,
    created_at
)
SELECT
    cm.id,    cm.conversation_id,
    cm.sender_id,
    'text',
    COALESCE(NULLIF(BTRIM(cm.message), ''), 'Message from previous chat'),
    cm.created_at
FROM legacy_agro_shield.chat_messages cm
JOIN conversations c ON c.id = cm.conversation_id
JOIN users u ON u.id = cm.sender_id
ON CONFLICT (id) DO NOTHING;

SELECT setval(pg_get_serial_sequence('chat_messages','id'),COALESCE((SELECT MAX(id) FROM chat_messages),0)+1,FALSE);

-- ---------------------------------------------------------------------------
-- Notifications
-- ---------------------------------------------------------------------------

INSERT INTO notifications (
    id,
    user_id,
    notification_type,
    title,
    message,
    is_read,
    created_at
)
SELECT
    n.id,    n.user_id,
    CASE
        WHEN n.type = 'chat_message' THEN 'message'
        WHEN n.type = 'wallet_deposit_expired' THEN 'payment'
        WHEN n.type IN ('negotiation', 'negotiation_message', 'negotiation_offer') THEN 'marketplace'
        ELSE 'system'
    END,
    COALESCE(NULLIF(BTRIM(n.title), ''), 'Agro-Shield update'),
    COALESCE(NULLIF(BTRIM(n.message), ''), 'You have a new Agro-Shield update.'),
    COALESCE(n.is_read, FALSE),
    n.created_at
FROM legacy_agro_shield.notifications n
JOIN users u ON u.id = n.user_id
ON CONFLICT (id) DO NOTHING;

SELECT setval(pg_get_serial_sequence('notifications','id'),COALESCE((SELECT MAX(id) FROM notifications),0)+1,FALSE);

-- ---------------------------------------------------------------------------
-- Diagnoses
-- ---------------------------------------------------------------------------

INSERT INTO diagnoses (
    id,
    user_id,
    farming_type_id,
    category,
    symptoms,
    result,
    confidence,
    created_at,
    updated_at
)
SELECT
    d.id,
    d.farmer_id,
    ft.id,
    NULLIF(BTRIM(d.category), ''),
    NULLIF(BTRIM(d.description), ''),
    NULLIF(BTRIM(d.result), ''),
    NULL,
    d.created_at,
    d.created_at
FROM legacy_agro_shield.diagnoses d
JOIN users u ON u.id = d.farmer_id
LEFT JOIN farming_types ft ON LOWER(ft.name) = LOWER(d.category)
    OR LOWER(ft.slug) = LOWER(d.category)
    OR (LOWER(d.category) = 'crops' AND ft.slug = 'crop')
ON CONFLICT (id) DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('diagnoses', 'id'),
    COALESCE((SELECT MAX(id) FROM diagnoses), 0) + 1,
    FALSE
);

INSERT INTO diagnosis_media (diagnosis_id, media_type, media_url, image_name, created_at)
SELECT
    dm.diagnosis_id,
    CASE
        WHEN LOWER(BTRIM(dm.media_type)) IN ('image', 'video', 'audio', 'document') THEN LOWER(BTRIM(dm.media_type))
        ELSE 'image'
    END,
    COALESCE(NULLIF(BTRIM(dm.file_name), ''), 'legacy-diagnosis-media-' || dm.id),
    NULLIF(BTRIM(dm.file_name), ''),
    dm.created_at
FROM legacy_agro_shield.diagnosis_media dm
JOIN diagnoses d ON d.id = dm.diagnosis_id
WHERE NOT EXISTS (
    SELECT 1
    FROM diagnosis_media current_media
    WHERE current_media.diagnosis_id = dm.diagnosis_id
      AND current_media.media_url = dm.file_name
);

-- ---------------------------------------------------------------------------
-- Marketplace analytics and search history
-- ---------------------------------------------------------------------------

INSERT INTO market_events (event_type, product_id, user_id, session_id, metadata, created_at)
SELECT
    COALESCE(NULLIF(BTRIM(me.event_type), ''), 'legacy_event'),
    CASE WHEN EXISTS (SELECT 1 FROM products p WHERE p.id = me.crop_id) THEN me.crop_id END,
    CASE WHEN EXISTS (SELECT 1 FROM users u WHERE u.id = me.user_id) THEN me.user_id END,
    me.session_id,
    COALESCE(me.metadata, '{}'::jsonb),
    me.created_at
FROM legacy_agro_shield.market_events me
WHERE NOT EXISTS (
    SELECT 1
    FROM market_events current_event
    WHERE current_event.event_type = me.event_type
      AND current_event.created_at = me.created_at
      AND current_event.session_id IS NOT DISTINCT FROM me.session_id
);

INSERT INTO marketplace_searches (search_term, normalized_produce, location, user_id, session_id, created_at)
SELECT
    ms.search_term,
    ms.normalized_produce,
    ms.location,
    CASE WHEN EXISTS (SELECT 1 FROM users u WHERE u.id = ms.user_id) THEN ms.user_id END,
    ms.session_id,
    ms.created_at
FROM legacy_agro_shield.marketplace_searches ms
WHERE NOT EXISTS (
    SELECT 1
    FROM marketplace_searches current_search
    WHERE current_search.search_term = ms.search_term
      AND current_search.created_at = ms.created_at
      AND current_search.session_id IS NOT DISTINCT FROM ms.session_id
);

INSERT INTO external_market_prices (
    produce_type,
    price_per_unit,
    unit,
    currency,
    provider,
    market_name,
    lga,
    state,
    country,
    collected_at,
    imported_at,
    source_url
)
SELECT
    emp.produce_type,
    GREATEST(COALESCE(emp.price_per_unit, 0), 0),
    COALESCE(NULLIF(BTRIM(emp.unit), ''), 'unit'),
    COALESCE(NULLIF(BTRIM(emp.currency), ''), 'NGN'),
    COALESCE(NULLIF(BTRIM(emp.provider), ''), 'Legacy Agro-Shield'),
    COALESCE(NULLIF(BTRIM(emp.market_name), ''), 'Unknown market'),
    emp.lga,
    emp.state,
    COALESCE(NULLIF(BTRIM(emp.country), ''), 'Nigeria'),
    emp.collected_at,
    COALESCE(emp.imported_at, emp.collected_at),
    emp.source_url
FROM legacy_agro_shield.external_market_prices emp
WHERE NOT EXISTS (
    SELECT 1
    FROM external_market_prices current_price
    WHERE current_price.produce_type = emp.produce_type
      AND current_price.market_name = emp.market_name
      AND current_price.collected_at = emp.collected_at
);

INSERT INTO lga_neighbors (lga_id, neighbor_lga_id, created_at)
SELECT
    l1.id,
    l2.id,
    CURRENT_TIMESTAMP
FROM legacy_agro_shield.lga_neighbors ln
JOIN states s ON LOWER(BTRIM(s.name)) = LOWER(BTRIM(ln.state))
JOIN lgas l1 ON l1.state_id = s.id AND LOWER(BTRIM(l1.name)) = LOWER(BTRIM(ln.lga))
JOIN lgas l2 ON l2.state_id = s.id AND LOWER(BTRIM(l2.name)) = LOWER(BTRIM(ln.neighbor_lga))
ON CONFLICT (lga_id, neighbor_lga_id) DO NOTHING;

-- ---------------------------------------------------------------------------
-- Wallets and wallet history
-- ---------------------------------------------------------------------------

-- Preserve the legacy wallet number when it is unique. If the old data has a
-- duplicate or blank number, the wallet-number trigger assigns a new one.
WITH legacy_wallets AS (
    SELECT
        lw.farmer_id,
        COALESCE(NULLIF(BTRIM(lw.currency), ''), 'NGN') AS currency,
        GREATEST(COALESCE(lw.balance, 0), 0) AS balance,
        lw.created_at,
        COALESCE(lw.updated_at, lw.created_at) AS updated_at,
        NULLIF(BTRIM(f.wallet_number), '') AS legacy_wallet_number,
        COUNT(*) OVER (
            PARTITION BY NULLIF(BTRIM(f.wallet_number), '')
        ) AS wallet_number_uses
    FROM legacy_agro_shield.wallets lw
    JOIN legacy_agro_shield.farmers f ON f.id = lw.farmer_id
    JOIN users u ON u.id = lw.farmer_id
)
INSERT INTO wallets (
    owner_type,
    owner_id,
    currency,
    balance,
    wallet_number,
    created_at,
    updated_at
)
SELECT
    'user',
    farmer_id,
    currency,
    balance,
    CASE WHEN wallet_number_uses = 1 THEN legacy_wallet_number END,
    COALESCE(created_at, CURRENT_TIMESTAMP),
    COALESCE(updated_at, created_at, CURRENT_TIMESTAMP)
FROM legacy_wallets
ON CONFLICT (owner_type, owner_id, currency) DO UPDATE SET
    balance = EXCLUDED.balance,
    wallet_number = EXCLUDED.wallet_number,
    created_at = EXCLUDED.created_at,
    updated_at = EXCLUDED.updated_at;

INSERT INTO wallets (owner_type, owner_id, currency)
SELECT 'user', u.id, 'NGN'
FROM users u
ON CONFLICT (owner_type, owner_id, currency) DO NOTHING;

INSERT INTO wallet_transactions (wallet_id, type, amount, reference, created_at)
SELECT
    w.id,
    lt.type,
    GREATEST(lt.amount, 0.01),
    COALESCE(NULLIF(BTRIM(lt.reference), ''), 'legacy-wallet-txn-' || lt.id),
    lt.created_at
FROM legacy_agro_shield.wallet_transactions lt
JOIN legacy_agro_shield.wallets lw ON lw.id = lt.wallet_id
JOIN wallets w ON w.owner_type = 'user' AND w.owner_id = lw.farmer_id AND w.currency = lw.currency
ON CONFLICT (reference) DO NOTHING;

INSERT INTO wallet_deposits (
    wallet_id,
    provider,
    reference,
    provider_transaction_id,
    amount,
    currency,
    verified_amount,
    verified_currency,
    status,
    failure_reason,
    created_at,
    updated_at
)
SELECT
    w.id,
    COALESCE(NULLIF(BTRIM(ld.provider), ''), 'flutterwave'),
    ld.reference,
    ld.provider_transaction_id,
    GREATEST(ld.amount, 0.01),
    COALESCE(NULLIF(BTRIM(ld.currency), ''), 'NGN'),
    ld.verified_amount,
    ld.verified_currency,
    ld.status,
    ld.failure_reason,
    ld.created_at,
    COALESCE(ld.updated_at, ld.created_at)
FROM legacy_agro_shield.wallet_deposits ld
JOIN legacy_agro_shield.wallets lw ON lw.id = ld.wallet_id
JOIN wallets w ON w.owner_type = 'user' AND w.owner_id = lw.farmer_id AND w.currency = lw.currency
WHERE ld.status IN ('pending', 'processing', 'successful', 'failed', 'expired', 'reversed')
ON CONFLICT DO NOTHING;

-- Original order payments are historical records. Importing them must not
-- credit wallets again: the original balance and ledger already include them.
INSERT INTO transactions(order_id,buyer_id,seller_type,seller_id,amount,currency,
    payment_method,payment_reference,payment_provider,provider_transaction_id,
    status,paid_at,created_at,updated_at)
SELECT lp.order_id,o.buyer_id,p.owner_type,p.owner_id,lp.amount,lp.currency,
    lp.provider,lp.reference,lp.provider,NULLIF(lp.transaction_id,''),
    CASE WHEN lp.status IN ('successful','success','paid') THEN 'successful'
         WHEN lp.status IN ('failed','cancelled','refunded') THEN lp.status ELSE 'pending' END,
    CASE WHEN lp.status IN ('successful','success','paid') THEN lp.updated_at END,
    lp.created_at,lp.updated_at
FROM legacy_agro_shield.payments lp
JOIN legacy_agro_shield.orders lo ON lo.id=lp.order_id
JOIN orders o ON o.id=lp.order_id
JOIN products p ON p.id=lo.crop_id
ON CONFLICT(payment_reference) DO NOTHING;

INSERT INTO webhook_events(provider,transaction_id,event,payload,processed_at,created_at)
SELECT 'flutterwave',transaction_id,event,payload,processed_at,created_at
FROM legacy_agro_shield.webhook_events
ON CONFLICT(provider,transaction_id,event) DO NOTHING;

-- Legacy courier names do not prove approval as a delivery partner. Keep the
-- recorded progress and addresses; unfinished deliveries can be assigned now.
INSERT INTO deliveries(id,order_id,pickup_address,destination_address,
    status,tracking_code,picked_up_at,delivered_at,created_at,updated_at)
SELECT d.id,d.order_id,c.location,d.delivery_address,
    CASE WHEN d.status IN ('picked_up','in_transit','delivered','cancelled') THEN d.status
         WHEN d.status='failed' THEN 'cancelled' ELSE 'pending' END,
    NULLIF(BTRIM(d.tracking_number),''),d.picked_up_at,d.delivered_at,d.created_at,d.updated_at
FROM legacy_agro_shield.deliveries d
JOIN orders o ON o.id=d.order_id
JOIN legacy_agro_shield.crops c ON c.id=d.crop_id
ON CONFLICT(order_id) DO NOTHING;
SELECT setval(pg_get_serial_sequence('deliveries','id'),COALESCE((SELECT MAX(id) FROM deliveries),0)+1,FALSE);

-- Feedback is a purchase review only when exactly one delivered purchase
-- supports it. Other original feedback remains in the archive for review.
INSERT INTO product_reviews(product_id,reviewer_id,order_id,rating,comment,created_at)
SELECT f.product_id,f.user_id,matched.order_id,f.rating,NULLIF(BTRIM(f.comment),''),f.created_at
FROM legacy_agro_shield.feedback f
JOIN LATERAL (
    SELECT min(o.id) AS order_id FROM orders o JOIN order_items oi ON oi.order_id=o.id
    WHERE o.buyer_id=f.user_id AND oi.product_id=f.product_id AND o.status='delivered'
      AND o.created_at<=f.created_at HAVING count(DISTINCT o.id)=1
) matched ON TRUE
WHERE matched.order_id IS NOT NULL
ON CONFLICT DO NOTHING;

INSERT INTO wallet_withdrawals (
    id,
    wallet_id,
    user_id,
    reference,
    provider_transfer_id,
    amount,
    currency,
    bank_name,
    bank_code,
    account_name,
    account_number,
    status,
    approved_by,
    reminded_at,
    name_check_status,
    name_check_resolved,
    name_check_detail,
    name_check_at,
    created_at,
    updated_at
)
SELECT
    lww.id,
    w.id,
    lww.farmer_id,
    lww.reference,
    lww.provider_transfer_id,
    lww.amount,
    COALESCE(NULLIF(BTRIM(lww.currency), ''), 'NGN'),
    lww.bank_name,
    lww.bank_code,
    lww.account_name,
    lww.account_number,
    CASE
        WHEN lww.status IN ('pending', 'processing', 'paid', 'rejected', 'failed', 'cancelled', 'expired') THEN lww.status
        ELSE 'pending'
    END,
    CASE WHEN EXISTS (SELECT 1 FROM users u WHERE u.id = lww.approved_by) THEN lww.approved_by END,
    lww.reminded_at,
    CASE WHEN lww.name_check_status IN ('', 'verified', 'mismatch', 'unverified') THEN lww.name_check_status ELSE '' END,
    COALESCE(lww.name_check_resolved, ''),
    COALESCE(lww.name_check_detail, ''),
    lww.name_check_at,
    lww.created_at,
    COALESCE(lww.updated_at, lww.created_at)
FROM legacy_agro_shield.wallet_withdrawals lww
JOIN legacy_agro_shield.wallets lw ON lw.id = lww.wallet_id
JOIN wallets w ON w.owner_type = 'user' AND w.owner_id = lw.farmer_id AND w.currency = lw.currency
WHERE lww.amount > 0
  AND BTRIM(lww.reference) <> ''
  AND BTRIM(lww.bank_name) <> ''
  AND BTRIM(lww.bank_code) <> ''
  AND BTRIM(lww.account_name) <> ''
  AND lww.account_number ~ '^[0-9]{10}$'
ON CONFLICT DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('wallet_withdrawals', 'id'),
    COALESCE((SELECT MAX(id) FROM wallet_withdrawals), 0) + 1,
    FALSE
);

INSERT INTO wallet_transfers (
    id,
    sender_wallet_id,
    sender_id,
    recipient_wallet_id,
    recipient_id,
    recipient_phone,
    reference,
    amount,
    currency,
    note,
    status,
    approved_by,
    reminded_at,
    created_at,
    updated_at
)
SELECT
    lwt.id,
    sender_wallet.id,
    lwt.sender_id,
    recipient_wallet.id,
    lwt.recipient_id,
    lwt.recipient_phone,
    lwt.reference,
    lwt.amount,
    COALESCE(NULLIF(BTRIM(lwt.currency), ''), 'NGN'),
    COALESCE(lwt.note, ''),
    CASE
        WHEN lwt.status IN ('pending', 'completed', 'rejected', 'cancelled', 'expired') THEN lwt.status
        ELSE 'pending'
    END,
    CASE WHEN EXISTS (SELECT 1 FROM users u WHERE u.id = lwt.approved_by) THEN lwt.approved_by END,
    lwt.reminded_at,
    lwt.created_at,
    COALESCE(lwt.updated_at, lwt.created_at)
FROM legacy_agro_shield.wallet_transfers lwt
JOIN legacy_agro_shield.wallets sender_legacy_wallet ON sender_legacy_wallet.id = lwt.sender_wallet_id
JOIN legacy_agro_shield.wallets recipient_legacy_wallet ON recipient_legacy_wallet.id = lwt.recipient_wallet_id
JOIN wallets sender_wallet
    ON sender_wallet.owner_type = 'user'
   AND sender_wallet.owner_id = sender_legacy_wallet.farmer_id
   AND sender_wallet.currency = sender_legacy_wallet.currency
JOIN wallets recipient_wallet
    ON recipient_wallet.owner_type = 'user'
   AND recipient_wallet.owner_id = recipient_legacy_wallet.farmer_id
   AND recipient_wallet.currency = recipient_legacy_wallet.currency
WHERE lwt.sender_id <> lwt.recipient_id
  AND lwt.amount > 0
  AND BTRIM(lwt.reference) <> ''
  AND BTRIM(lwt.recipient_phone) <> ''
ON CONFLICT DO NOTHING;

SELECT setval(
    pg_get_serial_sequence('wallet_transfers', 'id'),
    COALESCE((SELECT MAX(id) FROM wallet_transfers), 0) + 1,
    FALSE
);

INSERT INTO wallet_virtual_accounts (
    user_id,
    provider,
    tx_ref,
    order_ref,
    provider_ref,
    account_number,
    bank_name,
    account_name,
    currency,
    status,
    identity_kind,
    consent_version,
    consent_at,
    failure_reason,
    last_checked_at,
    created_at,
    updated_at
)
SELECT DISTINCT ON (lva.farmer_id)
    lva.farmer_id,
    COALESCE(NULLIF(BTRIM(lva.provider), ''), 'flutterwave'),
    COALESCE(NULLIF(BTRIM(lva.tx_ref), ''), 'legacy-va-' || lva.id),
    COALESCE(lva.order_ref, ''),
    COALESCE(lva.provider_ref, ''),
    CASE WHEN lva.account_number ~ '^[0-9]{10}$' THEN lva.account_number ELSE '' END,
    COALESCE(lva.bank_name, ''),
    COALESCE(lva.account_name, ''),
    COALESCE(NULLIF(BTRIM(lva.currency), ''), 'NGN'),
    CASE WHEN lva.status IN ('pending', 'active', 'failed') THEN lva.status ELSE 'failed' END,
    CASE WHEN lva.identity_kind IN ('bvn', 'nin') THEN lva.identity_kind ELSE '' END,
    COALESCE(lva.consent_version, ''),
    lva.consent_at,
    COALESCE(lva.failure_reason, ''),
    lva.last_checked_at,
    lva.created_at,
    COALESCE(lva.updated_at, lva.created_at)
FROM legacy_agro_shield.wallet_virtual_accounts lva
JOIN users u ON u.id = lva.farmer_id
ORDER BY lva.farmer_id, lva.created_at DESC
ON CONFLICT DO NOTHING;

INSERT INTO bank_account_lookups (account_number, bank_code, bank_name, account_name, probed_at)
SELECT
    bal.account_number,
    bal.bank_code,
    COALESCE(bal.bank_name, ''),
    COALESCE(bal.account_name, ''),
    COALESCE(bal.probed_at, CURRENT_TIMESTAMP)
FROM legacy_agro_shield.bank_account_lookups bal
WHERE bal.account_number ~ '^[0-9]{10}$'
ON CONFLICT (account_number, bank_code) DO NOTHING;

INSERT INTO platform_revenue (order_id, reference, gross_amount, amount, currency, rate_percent, created_at)
SELECT
    pr.order_id,
    pr.reference,
    pr.gross_amount,
    pr.amount,
    COALESCE(NULLIF(BTRIM(pr.currency), ''), 'NGN'),
    pr.rate_percent,
    pr.created_at
FROM legacy_agro_shield.platform_revenue pr
JOIN orders o ON o.id = pr.order_id
WHERE pr.gross_amount > 0
  AND pr.amount > 0
  AND pr.amount <= pr.gross_amount
ON CONFLICT DO NOTHING;
