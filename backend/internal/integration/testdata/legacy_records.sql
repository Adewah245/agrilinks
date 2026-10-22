-- Synthetic records only. IDs are deliberately non-sequential.
INSERT INTO farmers(id,full_name,phone,email,password_hash,location,lga,state,country,role,wallet_number,bank_name,bank_code,account_name,account_number)
VALUES(11,'Legacy seller','08010000011','seller@example.test','test-hash','Otukpo','Otukpo','Benue','Nigeria','farmer','8000000011','Test bank','001','Legacy seller','0123456789'),
      (21,'Legacy buyer','2348010000021','buyer@example.test','test-hash','Makurdi','Makurdi','Benue','Nigeria','buyer','8000000021',NULL,NULL,NULL,NULL);
INSERT INTO wallets(id,farmer_id,balance,currency) VALUES(31,11,1234.56,'NGN'),(32,21,765.44,'NGN');
INSERT INTO crops(id,farmer_id,name,quantity,unit,location,lga,state,country,price_per_unit,listed_for_sale)
VALUES(41,11,'Legacy maize',10,'kg','Otukpo','Otukpo','Benue','Nigeria',90,TRUE);
INSERT INTO crop_images(crop_id,image_url,is_primary) VALUES(41,'/static/test-maize.jpg',TRUE);
INSERT INTO produce_price_history(crop_id,price_per_unit,unit) VALUES(41,90,'kg');
INSERT INTO inventory_history(crop_id,quantity_before,quantity_change,quantity_after,reason) VALUES(41,12,-2,10,'sold');
INSERT INTO orders(id,buyer_id,crop_id,quantity,total_price,status) VALUES(51,21,41,2,180,'delivered');
INSERT INTO payments(order_id,reference,transaction_id,amount,currency,status) VALUES(51,'old-order-payment','123456',180,'NGN','successful');
INSERT INTO deliveries(id,order_id,buyer_id,farmer_id,crop_id,quantity,delivery_address,country,status,tracking_number,picked_up_at,delivered_at)
VALUES(61,51,21,11,41,2,'Buyer address','Nigeria','delivered','old-tracking-code',CURRENT_TIMESTAMP-INTERVAL '2 hours',CURRENT_TIMESTAMP-INTERVAL '1 hour');
INSERT INTO feedback(user_id,farmer_id,product_id,rating,comment) VALUES(21,11,41,4,'Good maize');
INSERT INTO negotiations(id,crop_id,buyer_id,farmer_id,quantity,initial_offer_price,current_offer_price,round_count,max_rounds,status,expires_at)
VALUES(71,41,21,11,1,70,80,2,5,'open',CURRENT_TIMESTAMP+INTERVAL '1 day');
INSERT INTO negotiation_messages(id,negotiation_id,sender_id,message_type,offer_price,message,offer_status)
VALUES(81,71,21,'offer',70,'Can you sell for this price?','pending'),(82,71,11,'offer',80,'My best price','pending');
INSERT INTO cart_items(id,buyer_id,crop_id,negotiation_id,quantity,price_per_unit,total_price) VALUES(91,21,41,71,1,80,80);
INSERT INTO conversations(id,name,type,created_by) VALUES(101,'Old farming room','group',11);
INSERT INTO conversation_members(conversation_id,user_id) VALUES(101,11),(101,21);
INSERT INTO chat_messages(id,conversation_id,sender_id,message,created_at)
VALUES(111,101,11,'First saved message','2025-01-01 12:00:00'),(112,101,11,'Another message at the same time','2025-01-01 12:00:00');
INSERT INTO notifications(user_id,title,message,type) VALUES(21,'Order arrived','Your maize arrived','order');
INSERT INTO diagnoses(id,farmer_id,category,description,result) VALUES(121,11,'crop','Leaves turning yellow','Saved diagnosis from the old app');
INSERT INTO diagnosis_media(diagnosis_id,media_type,file_name) VALUES(121,'image','/uploads/test-leaf.jpg');
INSERT INTO market_events(event_type,crop_id,user_id,session_id) VALUES('product_view',41,21,'test-session');
INSERT INTO marketplace_searches(search_term,normalized_produce,user_id) VALUES('Maize','maize',21);
INSERT INTO external_market_prices(produce_type,price_per_unit,unit,currency,provider,market_name,state,lga,country,collected_at)
VALUES('maize',90,'kg','NGN','Test price provider','Test market','Benue','Otukpo','Nigeria',CURRENT_TIMESTAMP);
INSERT INTO lga_neighbors(state,lga,neighbor_lga) VALUES('Benue','Otukpo','Ohimini');
INSERT INTO wallet_transactions(wallet_id,reference,type,amount) VALUES(31,'old-deposit','deposit',1234.56),(32,'buyer-deposit','deposit',765.44);
INSERT INTO wallet_deposits(wallet_id,reference,provider_transaction_id,amount,currency,verified_amount,verified_currency,status)
VALUES(31,'old-deposit','9876',1234.56,'NGN',1234.56,'NGN','successful');
INSERT INTO wallet_withdrawals(wallet_id,farmer_id,reference,amount,currency,bank_name,bank_code,account_name,account_number,status)
VALUES(31,11,'old-withdrawal',10,'NGN','Test bank','001','Legacy seller','0123456789','cancelled');
INSERT INTO wallet_transfers(sender_wallet_id,sender_id,recipient_wallet_id,recipient_id,recipient_phone,reference,amount,currency,status)
VALUES(31,11,32,21,'2348010000021','old-transfer',10,'NGN','cancelled');
INSERT INTO wallet_virtual_accounts(farmer_id,tx_ref,account_number,bank_name,account_name,status,identity_kind,consent_version,consent_at)
VALUES(11,'old-virtual-account','1234567890','Test bank','Legacy seller','active','nin','test-consent',CURRENT_TIMESTAMP);
INSERT INTO bank_account_lookups(account_number,bank_code,bank_name,account_name) VALUES('0123456789','001','Test bank','Legacy seller');
INSERT INTO platform_revenue(order_id,reference,gross_amount,amount,currency,rate_percent) VALUES(51,'old-platform-fee',180,1.80,'NGN',1);
INSERT INTO webhook_events(transaction_id,event,payload,processed_at) VALUES('123456','charge.completed','{}',CURRENT_TIMESTAMP);
