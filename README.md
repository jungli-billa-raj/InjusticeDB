# InjusticeDB
Collection of all the injustices in India. 

People forget. That's what the most frustrating part. Maybe most don't care. And this platform is **NOT** for them. 
I'm thinking of creating a DB(Database) for people to upload injustices and assets related to them here. Most importantly, the volunteers are suggested to fill in a range of current details about the injustice like:
1. Severity of Crime (1 to 10)
2. Culprit Status (suspect, accused, guilty, convicted)
3. Justice Status (proceeding, served)
4. Culprit Details (Culprit could be an individual or an organization)
5. Assets ( Images, Videos, Articles' links)
6. Obviously, the Full Story
etc...

What good is this gonna do anyways? I don't know. I'm frustrated and angry as hell. The lack of accountability is perhaps the main point of anger. Goons in civil dresses are thrashing Students who are exercising their Constitutional Rights. The people tasked with enforcing the rights are roaming around thrashing students left and right, without wearing their name tags. Authorites are breaking bones, bleeding 12 year olds and sexually harrassing girls. To instill fear in you. I saw a clip from this protest where around 30 "policemen" circled around a shirtless dude whose shirt probably was torn off, thrashed him and made him go inside a bus followed by 5 other "policemen".They know nothing is gonna happen to them. It's infuriating. My anger from the feeling of entrapment and helplessness knows no bounds. If you still think that the protesters deserved all this, then you are a part of the problem. Can having a unified collection of all such injustices solve the problem. Definitely not. In fact, I wouldn't take you seriously if you said anything ending with ".... this is going to solve the problem". 
In my opinion, the internet has made lives of a few culprits living hell in the past. Take the case of [RaviBhatia](https://www.vice.com/en/article/how-can-she-slap-india-viral-memes/) who has made a successful career for himself in the entertainment industry while the guy who wronged him is selling cheap chinese spin off products, all thanks to the guy who posted the video of this incident on YouTube. There are other cases where people still publically shame the culprit on their socials to this day, proving that the Internet DOES remember. Maybe, the internet, you and me, can help treat this national level Dementia.

Other use cases could be:
1. Checking for updates on a crime.
2. To read the full story. 
3. Fix misleading narrative.
4. Discuss 



In addition to this prime functionality of this platform, I'm considering other features. 
**Keep in mind that this is a dynamic project. If you have suggestions on functionalities to add or approaches to take, I'll gladly include them. Or if you know programming, then raise a PR. People registered on the platform will have the privilege to vote on matters of changes in this dynamic platform. I'm working for a vision that I don't have. I just want the daily unconstitutional proceedings to stop.**

## YDCIDC movement 
**You Don't Care, I don't Care** 
This is primarily targeted towards celebrities. Another affluent and powerful class of people that would be nothing without us. If they don't want to do anything when their audience's constitutional rights are blatantly and shamelessly violated, why are we funding their extravagant and carefree lifestyle with our time, attention and money. Teenagers and adults alike, love Bollywood-Tollywood dramas, Cricket, Comedy, Singers, Infleuncers etc. Where are they when **YOU** need **thier** support?????

I'm not afraid to call names. Where is Kohli? Where is Sharma? Where is Kartik Aryan? Where is Akshay Kumar? In foreign. Or in an ultra gated society in India that's only physically in India. I want to ask you. Why thefkkkk are you making memes about them? Raju from Hera-Pheri is not real. Why the fkkk are u making edits of "Sexy Sixes by Kohli" and why the fkk are you obssessed who Kartik Aryan is dating? 
Tell them in the comments of their next tweet, or IG post, "You don't care about me, I don't fking care about u". 
A country is a big family. If you act like the nicest person in the room just to fufil your needs, then sorry, you are not a member.

### I've no more ideas. If you have then let's discuss them. 

---

## Features

- **Crime Data collection with updates:** 
- **Highlighting People Who Couldn't Care Less About The Common Indians**
- **Requesting, Discussing and Voting for new Features/Sections** 

---

## Contribution Guide
**InjusticeDB** is built on the principle that transparency, immutability, and crowdsourced truth can empower real-world accountability. Every line of code, bug fix, and documentation improvement moves the platform forward.

This document outlines the workflow, architecture guidelines, and specifically the nuances of our **Crowdsourced Verification System**.

## 📐 Architecture & Principles

InjusticeDB follows **Hexagonal Architecture** (Ports and Adapters) in Go:

1. **Interfaces First:** All database capabilities are defined in `internal/db/interfaces.go`. Endpoints write to interfaces, not concrete PostgreSQL structs.
2. **Database Integrity:** PostgreSQL is our single source of truth. Schema changes require explicit up/down migration files in `migrations/`. I'm using [golang-migrate](https://github.com/golang-migrate/migrate).
3. **Immutability & Soft Deletes:** Historical reports and evidence assets are never hard-deleted immediately. Evidence assets utilize a 30-day soft-deletion grace period prior to permanent cleanup.
4. **Wayback Machine Backup:** Every article linked as assets in an `incident` will eventually be locked in through the Wayback Machine. I'm still in process of figuring out the cost and if this is even needed right now.

---

## ⚖️ Understanding the Verification System
### 1. Dual-Status Architecture

The post is owned by no one, not even the dude who posted it. Every incident tracks two distinct state variables:

* **`VerificationStatus`** (`pending` $\rightarrow$ `verified` / `rejected`): Driven purely by crowdsourced community voting and weighted credibility. Every new account starts with credibility score of 100. They increase and decrease when they changes they made are added or removed by the community. 
* **`JusticeStatus`** (`proceeding` $\rightarrow$ `served` / `stalled`): Driven by real-world legal outcomes and official documentation updates.

### 2. Weighted Crowd Voting (`VerificationRepository`)

When a user casts a vote via `CastVote(ctx, incidentID, userID, vote)`:

* **Vote Types:** Must be typed using `models.VoteType` (`models.VoteVerify` or `models.VoteReject`). 
* **Idempotency & Updates:** If a user votes `verify` and later changes their mind to `reject`, the database updates their existing vote rather than incrementing a duplicate row.
* **Credibility Weighting:** A user's vote carries weight proportional to their `credibility_score` (managed via `UserRepository.UpdateCredibility`). Users with higher verified platform contributions have stronger weight in moving an incident from `pending` to `verified`.

### 3. Git-Style Revision Control vs. Voting

Voting does **not** modify the underlying incident text. If new evidence emerges:

1. A new `IncidentRevision` is created with an incremented version number (`v1` $\rightarrow$ `v2`).
2. Major revisions trigger a re-evaluation of the verification tally to ensure the new facts are re-vetted by the community.

So, every `incident` will have a version number. If the user feel like some `incident` is lying or pushing an agenda, they can check out previous versions or make changes to create a new version. 

---

## 🛠️ Local Development & Testing Workflow

### 1. Prerequisites

* **Go 1.26.5+**
* **Docker & Docker Compose** (for local PostgreSQL)
* **golang-migrate** CLI

### 2. Setting Up the Database

Start local PostgreSQL and run migrations:

```bash
# Start Postgres container
docker-compose up -d

# Run migrations up
migrate -path=./migrations -database="postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable" up

```

### 3. Running Test Suites

Before submitting a Pull Request, all unit and integration tests **must** pass cleanly:

```bash
go test -v ./internal/db

```

*Note: Integration tests target local PostgreSQL. Ensure your local database container is running.*

---

## 📬 Submitting a Pull Request (PR)

1. **Fork the Repository** and create your branch from `main`:
```bash
git checkout -b feature/my-cool-feature

```


2. **Keep Commit Messages Clear:** Use descriptive prefixes (`feat:`, `fix:`, `docs:`, `refactor:`).
3. **Include Tests:** Any new repository method or handler must include corresponding test cases in `*_test.go`.
4. **Open a PR:** Describe *what* changes were made and *why*. Link any relevant issues.

---

## ❤️ Welcome aboard!

Thank you for your time. 
Welcome to the **InjusticeDB** community. Let's build this. 