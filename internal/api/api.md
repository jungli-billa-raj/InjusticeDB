
---

### 1. Users (`/api/v1/users`) [DONE]

* **`POST /api/v1/users/upsert`** (Protected)
* Handled by: `UserRepository.CreateOrUpdate`
* Creates or updates the user profile on login.


* **`GET /api/v1/users/{id}`** (Public / Protected)
* Handled by: `UserRepository.GetByID`
* Retrieves profile metadata and credibility score.


* **`PATCH /api/v1/users/{id}/credibility`** (Admin / Internal)
* Handled by: `UserRepository.UpdateCredibility`
* Adjusts credibility score delta (`+` or `-`).



---

### 2. Incidents & Revision History (`/api/v1/incidents`) [DONE]

* **`POST /api/v1/incidents`** (Protected)
* Handled by: `IncidentRepository.Create`
* Creates master record and seeds Version 1 revision.


* **`GET /api/v1/incidents/{id}`** (Public)
* Handled by: `IncidentRepository.GetByID`
* Retrieves the full latest snapshot of an incident.


* **`PATCH /api/v1/incidents/{id}/verification`** (Moderator / Admin)
* Handled by: `IncidentRepository.UpdateVerificationStatus`
* Updates verification status (`pending`, `verified`, `rejected`, `disputed`).



#### **Revision Version Control Sub-routes:** [DONE]

* **`POST /api/v1/incidents/{id}/revisions`** (Protected)
* Handled by: `IncidentRepository.CreateRevision`
* Submits an edit (bumps version number, logs change summary).


* **`GET /api/v1/incidents/{id}/revisions`** (Public)
* Handled by: `IncidentRepository.ListRevisions`
* Retrieves all historical revisions for an incident.


* **`GET /api/v1/incidents/{id}/revisions/{version}`** (Public)
* Handled by: `IncidentRepository.GetRevision`
* Retrieves a specific historical version snapshot.



---

### 3. Culprits & Entities (`/api/v1/people` & `/api/v1/incidents/{id}/culprits`) [DONE]

* **`POST /api/v1/people`** (Protected)
* Handled by: `CulpritRepository.CreatePerson`
* Registers a new individual/entity in the system.


* **`POST /api/v1/incidents/{id}/culprits`** (Protected)
* Handled by: `CulpritRepository.LinkToIncident`
* Links a person to an incident with a status (`suspect`, `accused`, `guilty`, `convicted`).


* **`GET /api/v1/incidents/{id}/culprits`** (Public)
* Handled by: `CulpritRepository.GetCulpritsForIncident`
* Retrieves all linked culprits for an incident.


* **`PATCH /api/v1/incidents/{id}/culprits/{person_id}`** (Protected / Moderator)
* Handled by: `CulpritRepository.UpdateCulpritStatus`
* Updates legal/culprit status for a linked person.



---

### 4. Verification Voting (`/api/v1/incidents/{id}/verifications`) [DONE]

* **`POST /api/v1/incidents/{id}/verifications`** (Protected)
* Handled by: `VerificationRepository.CastVote`
* User casts or updates their vote (`verify` / `reject`).


* **`GET /api/v1/incidents/{id}/verifications/tally`** (Public)
* Handled by: `VerificationRepository.GetVoteTally`
* Returns total verify and reject counts.



---

### 5. Assets & Media (`/api/v1/assets` & `/api/v1/incidents/{id}/assets`)

* **`POST /api/v1/incidents/{id}/assets`** (Protected)
* Handled by: `AssetRepository.AddAssets`
* Attaches evidence links or media files to an incident.


* **`GET /api/v1/incidents/{id}/assets`** (Public)
* Handled by: `AssetRepository.GetByIncidentID`
* Lists active (non-soft-deleted) assets.


* **`PATCH /api/v1/assets/{id}/archive`** (Worker / System)
* Handled by: `AssetRepository.UpdateArchiveURL`
* Attaches background archive link to an asset.


* **`DELETE /api/v1/assets/{id}`** (Protected / Owner)
* Handled by: `AssetRepository.SoftDeleteAsset`
* Soft-deletes an asset.


* **`POST /api/v1/assets/{id}/restore`** (Protected / Admin)
* Handled by: `AssetRepository.RestoreAsset`
* Restores a soft-deleted asset.


* **`DELETE /api/v1/assets/cleanup`** (System Cron Job)
* Handled by: `AssetRepository.HardDeleteExpiredAssets`
* Permanently purges assets soft-deleted past the cutoff (e.g. 30 days).



---

### 6. Comments & Discussions (`/api/v1/incidents/{id}/comments`)

* **`POST /api/v1/incidents/{id}/comments`** (Protected)
* Handled by: `CommentRepository.CreateComment`
* Posts a top-level comment or nested reply (`parent_id`).


* **`GET /api/v1/incidents/{id}/comments`** (Public)
* Handled by: `CommentRepository.ListCommentsByIncident`
* Returns threaded, hierarchical comments with replies.



---

### 7. Private Messaging DMs (`/api/v1/conversations`)

* **`POST /api/v1/conversations`** (Protected)
* Handled by: `MessagingRepository.GetOrCreateConversation`
* Gets or creates a canonical DM space between the requester and target user.


* **`GET /api/v1/conversations`** (Protected)
* Handled by: `MessagingRepository.ListConversations`
* Lists active conversations for the authenticated user.


* **`POST /api/v1/conversations/{id}/messages`** (Protected)
* Handled by: `MessagingRepository.SendMessage`
* Sends a direct message inside a conversation (enforced by RLS).


* **`GET /api/v1/conversations/{id}/messages`** (Protected)
* Handled by: `MessagingRepository.GetMessages`
* Fetches paginated messages for a conversation (enforced by RLS).



---

### 8. Target Public Registry (`/api/v1/targets`)

* **`POST /api/v1/targets`** (Protected)
* Handled by: `TargetRepository.CreateTarget`
* Adds an entry to the public target registry.


* **`GET /api/v1/targets`** (Public)
* Handled by: `TargetRepository.ListTargets`
* Lists registered targets with pagination.



---
