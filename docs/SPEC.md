# Upstore — Product Specification

> Shopify-like e-commerce platform built for Russian and Belarusian merchants.

---

## 1. Vision

**Tagline:** «Открой интернет-магазин за один день»  
**Core promise:** Launch a fully operational online store without a programmer or agency — in under 24 hours.

Upstore fills the gap left by Western platforms (Shopify, WooCommerce, BigCommerce) that are inaccessible or severely limited in Russia and Belarus due to payment infrastructure incompatibilities and sanctions. It is a fully localised, compliant, zero-commission alternative.

---

## 2. Target Market

- **Primary:** Small and mid-size Russian/Belarusian merchants wanting to sell online quickly
- **Geography:** Russia, Belarus
- **Language:** Russian
- **Support timezone:** UTC+3, via Telegram and phone (live humans, no chatbot)

---

## 3. Key Differentiators

| Feature | Detail |
|---|---|
| 0% sales commission | Platform revenue is subscription-only, no per-transaction cut |
| Local payments | СБП, ЮКасса, Тинькофф, Сбербанк — out of the box |
| Local delivery | СДЭК, Почта России, Boxberry — integrated with rate calc and tracking |
| Russian-language support | Telegram + phone, UTC+3, real staff |
| Data residency | All data stored on Russian servers (152-ФЗ compliant) |
| No card to start | 30-day free trial, email-only signup |
| One-click cancellation | No lock-in |

---

## 4. Pricing

| Plan | Price | Limits | Key Features |
|---|---|---|---|
| **Старт** | ₽1,490 / month | Up to 100 products | All payment systems, basic shipping |
| **Рост** | ₽3,990 / month | Up to 10,000 products | Analytics, promo codes, priority support |
| **Про** | ₽9,990 / month | Unlimited products | Multiple stores, API access, 1C integration, dedicated manager |

---

## 5. Core Feature Set

### 5.1 Merchant Onboarding
- Email-only registration (no card, no documents)
- Store setup wizard
- 30-day free trial on any plan

### 5.2 Product Catalog
- Manual product creation (photo, title, price, description)
- Bulk import via Excel
- Category and tag management

### 5.3 Storefront
- Mobile-friendly, responsive storefronts
- Customisable store appearance
- Public product pages with SEO-friendly URLs

### 5.4 Payments
- **Launch:** ЮКасса, СБП
- **Roadmap:** Тинькофф, Сбербанк
- 0% platform commission on all transactions

### 5.5 Shipping
- СДЭК, Почта России, Boxberry
- Automatic rate calculation at checkout
- Order tracking in merchant dashboard

### 5.6 Orders & Dashboard
- Order management and status updates
- Real-time sales analytics
- Conversion rate tracking
- Abandoned cart tracking

### 5.7 Marketing Tools
- Discount codes and promo codes
- Auto-application rules

### 5.8 Pro / Enterprise
- Multiple stores under one account
- REST API access
- 1C accounting system integration
- Dedicated account manager

---

## 6. Tech Stack

| Layer | Choice | Notes |
|---|---|---|
| Backend | Go | REST API; Fiber or Chi |
| Frontend (admin) | React + TypeScript + Tailwind | Merchant dashboard |
| Frontend (storefront) | Next.js | Public-facing store pages, SSR for SEO |
| Database | PostgreSQL | Primary datastore |
| Auth | JWT + refresh tokens | Merchant auth |
| Hosting | TBD | Yandex Cloud / VK Cloud preferred for 152-ФЗ compliance |
| CI/CD | GitHub Actions | |

---

## 7. Compliance

- **152-ФЗ** — Russian Federal Law on Personal Data: all personal data of Russian citizens must be stored on Russian territory. Hosting must be on Russian infrastructure (Yandex Cloud, VK Cloud, or self-hosted).
- Payment providers must be licensed by the Bank of Russia.

---

## 8. MVP Scope

The MVP delivers a **thin end-to-end slice**: one merchant can register, build a store, list products, and get paid.

### MVP Milestone Breakdown

#### M1 — Foundation
- [ ] Repo structure and Go project scaffold
- [ ] PostgreSQL schema: merchants, stores, products, orders
- [ ] Merchant auth (register, login, JWT)
- [ ] Basic admin dashboard shell (React)

#### M2 — Store Builder
- [ ] Store creation and settings (name, slug, logo)
- [ ] Product CRUD (create, edit, delete, publish)
- [ ] Photo upload (S3-compatible storage)
- [ ] Public storefront with product listing and detail pages

#### M3 — Checkout & Payments
- [ ] Cart and checkout flow
- [ ] ЮКасса integration
- [ ] СБП integration
- [ ] Order creation and confirmation email

#### M4 — Orders & Shipping
- [ ] Order management in dashboard
- [ ] СДЭК integration (rate calc at checkout)
- [ ] Order status updates

#### M5 — Billing & Launch
- [ ] Subscription plans (Старт / Рост / Про)
- [ ] 30-day free trial logic
- [ ] Plan enforcement (product limits, feature gates)
- [ ] Basic sales analytics (revenue, order count)

---

## 9. Out of Scope for MVP

- Excel bulk import
- Promo codes / discounts
- Abandoned cart tracking
- Boxberry / Почта России integrations
- 1C integration
- Multiple stores per account
- Public API
- Тинькофф / Сбербанк payment integrations

---

## 10. Success Metrics (MVP)

| Metric | Target |
|---|---|
| Time to first store live | < 1 hour for a new merchant |
| Successful checkout rate | > 95% |
| Free trial → paid conversion | Measure and iterate |
