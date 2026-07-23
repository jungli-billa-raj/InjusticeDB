## 1. Complete API Route Map for InjusticeDB

Here is every endpoint grouped by responsibility, including method, path, authentication requirement, and purpose:

### 🔐 Authentication (`/api/v1/auth`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/auth/google` | Public | Exchange Google ID token for app JWT |

---

### 🚨 Incidents & Version History (`/api/v1/incidents`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/incidents` | Public | List incidents with filters (`state`, `city`, `status`, `limit`, `offset`) | // There should be a normal endpoint that just provides the latest version of an incident.
| `POST` | `/api/v1/incidents` | Protected | Create a new incident report (auto-creates Version 1) |
| `GET` | `/api/v1/incidents/{id}` | Public | Fetch master incident details by UUID | // What do you mean by Master Incident?
| `POST` | `/api/v1/incidents/{id}/revisions` | Protected | Submit a git-style revision edit (creates Version $N+1$) |
| `GET` | `/api/v1/incidents/{id}/revisions` | Public | Fetch complete version history tree for an incident | // What function is this gonna call? In the frontend, I'm planning to just give options to select any one version. Showing all the versions in a dedicated web page will be huge request. 
| `GET` | `/api/v1/incidents/{id}/revisions/{version}` | Public | Fetch a specific historical snapshot (`/v1`, `/v2`) | // Yeah, this is what I had in mind. 

---

### 📁 Evidence Assets & Archiving (`/api/v1/incidents/{id}/assets`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/incidents/{id}/assets` | Protected | Upload evidence media/links & queue Wayback archiving | // Can you tell me about the pricing of Wayback Machine. I don't want another service to pay for. If it's cheap then I would love to save the verfied article's webpages on my own end. 
| `GET` | `/api/v1/incidents/{id}/assets` | Public | Fetch verified media and active archive URLs | // Okay, so this will be called when a user opens an incident page. Right?
| `DELETE` | `/api/v1/assets/{asset_id}` | Protected | Soft-delete evidence asset (initiates 30-day grace period) |

---

### 🗳️ Crowdsourced Verification & Voting (`/api/v1/incidents/{id}/vote`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/incidents/{id}/vote` | Protected | Cast or update crowd vote (`verify` / `reject`) |
| `GET` | `/api/v1/incidents/{id}/tally` | Public | Get aggregate verification score & vote counts |

---

### 👤 Culprits / Suspect Registry (`/api/v1/culprits`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/culprits` | Protected | Register a person/entity in the system |
| `POST` | `/api/v1/incidents/{id}/culprits` | Protected | Link a culprit to a specific incident |
| `GET` | `/api/v1/incidents/{id}/culprits` | Public | List all suspect records linked to an incident |// Yeah, I'm guessing this will also be called when a user opens an incident's page. Is this normal or are we making too many endpoints call for just viewing one incident's page? Or is there something I am not understanding? 

---

### 💬 Discussion Threads (`/api/v1/incidents/{id}/comments`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `POST` | `/api/v1/incidents/{id}/comments` | Protected | Post a public comment on an incident report |
| `GET` | `/api/v1/incidents/{id}/comments` | Public | Fetch discussion thread with pagination |

---

### ✉️ Private Messaging (`/api/v1/messaging`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/conversations` | Protected | List active 1-on-1 conversations for current user |
| `POST` | `/api/v1/conversations` | Protected | Start or get a private chat with another user |
| `GET` | `/api/v1/conversations/{id}/messages` | Protected | Fetch paginated chat history |
| `POST` | `/api/v1/conversations/{id}/messages` | Protected | Send a private message (enforced via RLS) |

---

### 🎯 Public Watchlist Registry (`/api/v1/targets`)

| Method | Endpoint | Auth | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/targets` | Public | Fetch public registry watchlist |
| `POST` | `/api/v1/targets` | Protected | Add a entity entry to the watchlist |
