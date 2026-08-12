# API Documentation

## GET /

Returns homepage.

---

## GET /register

Registration page.

---

## POST /register

Registers a farmer.

Body

```json
{
    "full_name":"",
    "phone":"",
    "password":"",
    "location":""
}
```

---

## GET /login

Login page.

---

## POST /login

Authenticates user.

---

## GET /dashboard

Farmer dashboard.