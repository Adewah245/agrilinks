document.addEventListener("DOMContentLoaded", () => {
    /* ============================================================
       DOM ELEMENTS
       ============================================================ */
    const messagesContainer = document.getElementById("messages");
    const messageForm = document.querySelector(".message-form");
    const messageInput = document.getElementById("message-input");
    const conversationIdInput = messageForm?.querySelector('input[name="conversation_id"]');
    const typingStatusEl = document.getElementById("typing-status");

    if (!messageForm || !conversationIdInput) return;

    const conversationID = conversationIdInput.value;
    const currentUserID = window.AGRO_SHIELD_USER?.id || null; // Assume user identity is injected or inferred

    /* ============================================================
       WEBSOCKET STATE
       ============================================================ */
    let socket = null;
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;
    let reconnectTimeout = null;
    let typingTimeout = null;

    /* ============================================================
       INIT WEBSOCKET CONNECTION
       ============================================================ */
    function connectWebSocket() {
        const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
        const wsUrl = `${protocol}//${window.location.host}/ws/chat?conversation_id=${conversationID}`;

        socket = new WebSocket(wsUrl);

        socket.onopen = () => {
            console.log("[Chat WS] Connected to conversation:", conversationID);
            reconnectAttempts = 0;
        };

        socket.onmessage = (event) => {
            try {
                const payload = JSON.parse(event.data);
                handleIncomingEvent(payload);
            } catch (err) {
                console.error("[Chat WS] Error parsing incoming JSON:", err);
            }
        };

        socket.onerror = (error) => {
            console.error("[Chat WS] Connection error:", error);
        };

        socket.onclose = (event) => {
            console.warn("[Chat WS] Disconnected. Event code:", event.code);
            attemptReconnect();
        };
    }

    function attemptReconnect() {
        if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 10000); // Exponential backoff
            console.log(`[Chat WS] Reconnecting in ${delay}ms (Attempt ${reconnectAttempts})`);
            reconnectTimeout = setTimeout(connectWebSocket, delay);
        } else {
            console.error("[Chat WS] Max reconnection attempts reached. Fallback to standard HTTP forms.");
        }
    }

    /* ============================================================
       EVENT DISPATCHER
       ============================================================ */
    function handleIncomingEvent(event) {
        switch (event.type) {
            case "new_message":
                appendIncomingMessage(event.data);
                break;
            case "typing":
                handleTypingEvent(event.data);
                break;
            case "read_receipt":
                handleReadReceipt(event.data);
                break;
            default:
                console.warn("[Chat WS] Unknown event type:", event.type);
        }
    }

    /* ============================================================
       MESSAGE RENDERING
       ============================================================ */
    function appendIncomingMessage(msg) {
        // Clear empty chat state placeholder if present
        const emptyState = messagesContainer.querySelector(".empty-chat");
        if (emptyState) emptyState.remove();

        const isMine = msg.sender_id === currentUserID;
        const messageDiv = document.createElement("div");
        messageDiv.className = `message ${isMine ? "mine" : "theirs"}`;
        messageDiv.dataset.messageId = msg.id;

        const formattedTime = new Date(msg.created_at).toLocaleTimeString([], {
            hour: "2-digit",
            minute: "2-digit",
        });

        let htmlContent = "";

        // Render sender name/avatar for incoming group messages
        if (!isMine && msg.sender_name) {
            htmlContent += `
                <a class="sender" href="/profile?id=${msg.sender_id}">
                    ${
                        msg.sender_photo
                            ? `<img src="${msg.sender_photo}" alt="${msg.sender_name}">`
                            : `<span role="img" aria-label="Profile picture unavailable">👤</span>`
                    }
                    ${escapeHTML(msg.sender_name)}
                </a>
            `;
        }

        htmlContent += `
            <div class="message-bubble">${escapeHTML(msg.content)}</div>
            <div class="message-time">
                <time class="local-time" datetime="${msg.created_at}">${formattedTime}</time>
                ${
                    isMine
                        ? `<span class="message-receipt">${msg.seen ? "✓✓ Seen" : "✓ Sent"}</span>`
                        : ""
                }
            </div>
        `;

        messageDiv.innerHTML = htmlContent;
        messagesContainer.appendChild(messageDiv);
        scrollToBottom();
    }

    /* ============================================================
       TYPING INDICATOR
       ============================================================ */
    function handleTypingEvent(data) {
        if (!typingStatusEl || data.user_id === currentUserID) return;

        if (data.is_typing) {
            typingStatusEl.textContent = `${data.user_name || "Someone"} is typing...`;
        } else {
            typingStatusEl.textContent = "";
        }
    }

    function sendTypingSignal(isTyping) {
        if (socket && socket.readyState === WebSocket.OPEN) {
            socket.send(
                JSON.stringify({
                    type: "typing",
                    data: {
                        conversation_id: conversationID,
                        is_typing: isTyping,
                    },
                })
            );
        }
    }

    /* ============================================================
       READ RECEIPTS
       ============================================================ */
    function handleReadReceipt(data) {
        if (data.sender_id === currentUserID) {
            document.querySelectorAll(".message.mine .message-receipt").forEach((receipt) => {
                receipt.textContent = "✓✓ Seen";
            });
        }
    }

    /* ============================================================
       FORM SUBMISSION & EVENT LISTENERS
       ============================================================ */
    messageForm.addEventListener("submit", (e) => {
        const text = messageInput.value.trim();
        if (!text) return;

        // Use WebSockets if connection is healthy
        if (socket && socket.readyState === WebSocket.OPEN) {
            e.preventDefault();

            const payload = {
                type: "send_message",
                data: {
                    conversation_id: conversationID,
                    message: text,
                },
            };

            socket.send(JSON.stringify(payload));
            messageInput.value = "";
            sendTypingSignal(false);
        }
        // Fall back to HTTP POST form submission automatically if WS is closed
    });

    messageInput.addEventListener("input", () => {
        sendTypingSignal(true);

        clearTimeout(typingTimeout);
        typingTimeout = setTimeout(() => {
            sendTypingSignal(false);
        }, 2000);
    });

    /* ============================================================
       UTILITIES
       ============================================================ */
    function scrollToBottom() {
        if (messagesContainer) {
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }
    }

    function escapeHTML(str) {
        return str.replace(/[&<>'"]/g, (tag) => {
            const chars = {
                "&": "&amp;",
                "<": "&lt;",
                ">": "&gt;",
                "'": "&#39;",
                '"': "&quot;",
            };
            return chars[tag] || tag;
        });
    }

    // Connect on load
    connectWebSocket();
});