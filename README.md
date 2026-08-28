# EKA ID - Universal Digital Identity Platform

> **One person -> One EKA ID -> One verifiable digital identity**

---

## 1. What is EKA ID? (In Plain English)

Imagine an ID card for the digital world that protects your privacy:

- **You get one unique ID number**: For example, `EKA-7K4M-92PX`.
- **Zero personal details in your ID number**: Your name, birth date, and phone number are never hidden inside your ID code.
- **Dynamic QR Codes**: When someone needs to verify you, you generate a secure QR code that only lasts for a few minutes (e.g. 15 minutes).
- **You choose what they see**: You can choose to show only "Identity is Valid" and "Name", while keeping your phone number, birthday, and address completely private.
- **Companies must ask your permission**: If an employer or bank wants to verify your ID, they send a request to your dashboard. You click **Approve** or **Deny**.

> **Important Note**: EKA ID is an independent private platform. It is **not** Aadhaar, not a replacement for government IDs, and not an official state identity.

---

## 2. How to Run It (One Single Command)

### Option A: Using PowerShell
Open PowerShell in this folder and run:
```powershell
.\start.ps1
```

### Option B: Windows Double-Click
Double-click the file named **`start.bat`** in this folder.

### Option C: Using Docker directly
```bash
docker compose up -d
```

That is it! All 4 services (Database, Redis, Backend API, and Web Portal) will start automatically, and your browser will open to:
- **Web App**: http://localhost:3000
- **API Docs**: http://localhost:8080/api/v1/docs

---

## 3. Ready-to-Use Test Accounts

You can test every feature right away without typing passwords. On the [Sign In page](http://localhost:3000/login), you will find quick-access buttons, or use:

| Account Type | Email | Password | What You Can Do |
| :--- | :--- | :--- | :--- |
| **Normal User** (John Mathew) | `john.mathew@example.com` | `Password123!` | View your Digital ID Card, create QR codes, approve company requests |
| **System Admin** | `admin@eka.dev` | `Password123!` | View all registered users, suspend IDs, view audit trail, review duplicates |
| **Create Your Own ID** | Visit `/register` | Any password | Test the 3-step registration wizard (Demo OTP code: `123456`) |

---

## 4. Tour of the Web Pages

Once you open **http://localhost:3000**, here is what you can explore:

1. **Home Page (`/`)**: Overview of how the privacy system works.
2. **Registration (`/register`)**: Create a brand new identity in 3 steps with mock phone verification (`123456`).
3. **User Dashboard (`/dashboard`)**:
   - **My Identity**: View your active verification level and profile details.
   - **Digital Card**: An interactive ID card with photo and QR code. You can click to flip it, print it, or save as PDF.
   - **QR Studio**: Generate a temporary verification QR code with custom expiration and claim scopes.
   - **Consent Requests**: Review and Approve/Deny verification requests sent by outside organizations.
   - **Privacy Rights**: Clear explanation of how your data is protected.
4. **Public Verifier (`/verify`)**: Anyone can scan or paste a QR token here to see a green "Identity Verified" stamp with only the details you allowed.
5. **Organization Portal (`/org`)**: Pretend to be a company (like Acme Corp) and request verification for an EKA ID.
6. **Admin Console (`/admin`)**: Elevated dashboard to manage system status, view security logs, and resolve duplicate identity alerts.

---

## 5. Privacy Superpowers

- **No Stalker QR Codes**: Standard QR codes put your real phone number and address in plain text. EKA ID never does that. QR codes only contain a short-lived link that expires.
- **Owner Consent**: No company can look up your personal information without your explicit approval.
- **Audit Trail**: Every time an organization checks your ID or you sign in, an unchangeable audit log records who, what, and when.
- **Redacted Passwords**: Passwords and secret tokens are never saved in logs.

---

## 6. License & Commercial Rules

EKA ID uses the **Community & Commercial Entity License**:

- **For Individuals, Students & Hobbyists**: **100% Free**. You are welcome to view, run, study, and test the project for personal learning or non-commercial use.
- **For Companies & Businesses**: **Must contact the author for written permission or an enterprise license** before deploying or using this software in business operations or commercial products.

---

## 7. For Developers & Technical Architecture

If you want to look under the hood:

- **Backend**: Go 1.23 modular monolith (`services/api`)
  - Cryptographic Crockford Base32 ID generator: 1.1 trillion permutations
  - JWT authentication & rate limiting
  - Automatic fallback: Works with PostgreSQL or in-memory mode
- **Frontend**: Next.js 14, React, Tailwind CSS, TypeScript (`apps/web`)
- **Database**: PostgreSQL 15 (`database/migrations/` and `database/seeds/`)
- **Cache**: Redis 7
- **Documentation**:
  - [ARCHITECTURE.md](docs/ARCHITECTURE.md) - Deep architectural diagrams
  - [API.md](docs/API.md) - Complete REST endpoint catalog
  - [SECURITY.md](docs/SECURITY.md) - Threat models and encryption rationale
  - [openapi.yaml](docs/openapi.yaml) - OpenAPI 3.0 specification

### Running the automated test suite:
```powershell
docker run --rm -v "${PWD}/services/api:/app" -w /app golang:1.23-alpine go test -v ./...
```