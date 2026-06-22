# Psycho-Prisma — Senior Engineer Review

> Repository: https://github.com/sway-developer/psycho-prisma  
> Reviewed: 2026-06-22

---

## Project Summary

**"Призма"** is a desktop psychological assessment application, likely for military psychology units (conscript testing). It wraps a Next.js 14 web app inside Electron, using Prisma with SQLite as the database and Lucia for session-based auth. The stack is modern and well-chosen for the domain. The project is at a very early stage (~2 commits) with working structure but significant gaps in correctness, security, and UX.

**Tech stack:** Next.js 14 · Electron 33 · Prisma (SQLite) · Lucia auth · shadcn/ui + Radix UI · React Hook Form + Zod · TanStack Query + Table · Recharts · Zustand · xlsx

---

## Part 1 — Critical Bugs & Security Issues

### 1. `.env` committed to git
The `.env` file is in the repo root and appears to be tracked. This leaks the `DATABASE_URL` and any future secrets (API keys, signing secrets) into version history permanently.

**Fix:**
```bash
echo ".env" >> .gitignore
git rm --cached .env
git commit -m "chore: remove .env from tracking"
```
Document required vars in `.env.example`.

---

### 2. `prisma.exe` binary in the repository
A compiled binary is committed to git. This is a build artifact, not source code. It inflates the repo, can't be diffed, and is a supply-chain security risk (tampered binary).

**Fix:** Add to `.gitignore`, delete from git history, and document how to obtain it in the README:
```gitignore
prisma.exe
*.exe
```

---

### 3. `FormSubmission` and `TestSubmission` have no Prisma relations
```prisma
model TestSubmission {
  userId String   // ← orphaned string, no @relation
  testId String   // ← orphaned string, no @relation
  ...
}
```
Prisma will not enforce referential integrity. Deleting a `User` or `Test` leaves dangling submission records with no cascade. SQLite foreign key enforcement is also off by default.

**Fix:** Add explicit relations and cascade rules:
```prisma
model TestSubmission {
  id        String   @id @default(uuid())
  user      User     @relation(fields: [userId], references: [id], onDelete: Cascade)
  userId    String
  test      Test     @relation(fields: [testId], references: [id], onDelete: Cascade)
  testId    String
  summary   String
  submission String
  createdAt DateTime @default(now())
  updatedAt DateTime @updatedAt
}
```
Mirror this change in `FormSubmission`.

---

### 4. `signUp` accepts `any` — no input validation
```typescript
export async function signUp(data: any) {
  console.log(data.phoneNumber);   // logs PII
  ...
  await prisma.user.create({ data: { ...credentials } }); // unvalidated spread
}
```
Arbitrary fields from the client can be spread directly into `prisma.user.create`. An attacker can set `role: "admin"` or inject any other field by crafting a request.

**Fix:** Define and enforce a Zod schema, just like `signIn` does. Never spread `any` into a DB write.

---

### 5. PII logged to console in production
```typescript
console.log(data.phoneNumber);
console.log(userExists);
```
Phone numbers and full user objects are logged. In a military context this is especially problematic. Remove all `console.log` calls from server actions.

---

### 6. Non-null assertion with optional chain — contradictory pattern
```typescript
userId: user?.id!,
```
`user?.id` returns `string | undefined`, and `!` suppresses the TypeScript error. If `useSession()` returns `null` (unauthenticated request), this produces `undefined` silently and the DB insert either fails at runtime or stores `undefined` as the user ID.

**Fix:** Guard explicitly:
```typescript
const user = await getSession();
if (!user) throw new Error("Unauthenticated");
userId: user.id,
```

---

### 7. `recoveryQuestionAnswer` stores a question type, not an answer
The field has `@default("Фамилия")` — "Фамилия" means "surname" and is a question category, not a recovery answer. The field name and default value contradict each other. This is either a naming bug or a logic bug that would cause the recovery flow to silently accept wrong answers.

---

## Part 2 — Architecture & Code Quality Issues

### 8. `useSession` violates React hook naming convention
```typescript
// authentication.ts
export async function useSession() { ... }  // NOT a hook
```
The `use` prefix is reserved for React hooks. This function is a plain async server utility. ESLint's rules-of-hooks will flag any call site that isn't a component. Rename to `getSession()` or `validateSession()`.

---

### 9. Electron window loads `localhost:3000` directly
```javascript
window.loadURL("http://localhost:3000");
```
The window opens immediately while `next start` may still be initializing. There is no wait-on mechanism, no retry, and no error page if the server isn't ready. In production, the user sees a blank/error screen while Next.js boots.

**Fix:** Use `wait-on` before spawning the Electron window, or load a local static export instead of a live server.

---

### 10. Project name is `"a"` in `package.json`
```json
{ "name": "a" }
```
Should be `"prisma"` or `"psycho-prisma"` or the chosen product name. This affects Electron Builder's output naming and npm scripts.

---

### 11. All domain data stored as JSON strings in SQLite
`questions`, `scales`, `stanTable`, `tGradeTable`, `summaryTable` are `String` columns containing `JSON.stringify()`-ed data. This is forced by SQLite's lack of a native JSON type. While workable short-term, it means:
- No partial querying on question data
- No schema validation at the DB layer
- All parsing must happen in application code

**Recommendation:** Document this explicitly. If the app ever moves to PostgreSQL, migrate these to `JSONB` columns with proper schema types.

---

### 12. Missing Prisma enums for role, rank, servingKind
```prisma
role String      // "user" | "admin" | ???
rank String      // free text
servingKind String   // "Срочная служба" | ???
```
Using free strings means typos silently corrupt data. These should be Prisma `enum` types or at minimum validated by Zod schemas before writing.

---

### 13. Inconsistent ID generation
- `User.id`: generated manually in action code via `randomUUID().toString()`
- All other models: `@default(uuid())` at schema level

The `User` model should also use `@default(uuid())` in the schema, not caller-generated IDs. This removes an entire category of bugs where callers forget to supply the ID.

---

### 14. No pagination on list endpoints
`findAllUsers()` and `findAllTests()` appear to fetch all records with no `take`/`skip`. For a small unit this is harmless, but it is an architectural gap that causes performance problems if the database grows.

---

### 15. Missing `loading.tsx` and `error.tsx` files
The Next.js app folder has no `loading.tsx` or `error.tsx` at any route level. Server component suspense is not handled, so slow DB queries produce no loading feedback and errors bubble to a generic crash page.

---

### 16. `// @ts-expect-error` in server.cjs
```javascript
// @ts-expect-error
await nextBuild(path.join(__dirname, ".."));
```
This suppresses a legitimate type error. The `nextBuild` import path `next/dist/build/index.js` is internal Next.js API, not a public contract, and can break on Next.js minor updates. Use the Next.js CLI (`next build`) via `child_process.exec` instead.

---

### 17. Hardcoded window dimensions, no state persistence
```javascript
width: 1600, height: 900,
```
No minimum size, no `minWidth`/`minHeight`, no saving of window position or size between sessions. On a 1366×768 screen (common in institutional setups), the app overflows.

---

## Part 3 — Missing Features (High Priority)

| # | Feature | Reason |
|---|---------|--------|
| 1 | **PDF report export** | Core deliverable for psychological assessments — results need to be printed and filed |
| 2 | **Group-based test assignment** | Assign a test to a whole group at once instead of user-by-user |
| 3 | **Search & filter on all list pages** | Users list and test list need text search and column filters |
| 4 | **Test result visualization** | Radar/spider charts for multi-scale results using the already-imported Recharts library |
| 5 | **Score interpretation display** | The `stanTable`/`tGradeTable`/`summaryTable` data is uploaded but there is no UI to display computed interpretations |
| 6 | **Archive workflow** | `/dashboard/archive` directory exists but its purpose and implementation are undefined |
| 7 | **Database backup/restore** | SQLite file with no backup mechanism; critical for sensitive personnel data |
| 8 | **Input validation on forms (Zod)** | `signUp` and likely other actions lack schema validation |

---

## Part 4 — New Features (Roadmap)

### Near-term

1. **Bulk operations panel**  
   Select multiple users → assign test, export results, move group. Essential for working with conscript cohorts (50–200 people at a time).

2. **Timed test sessions**  
   Enforce `ttc` (time to complete) at runtime with a visible countdown. Currently the field exists in the model but is never enforced.

3. **Admin audit log**  
   Who created which user, who submitted which test, who changed a user's role. Especially important given military context.

4. **First-login forced password change**  
   The `firstTimer` boolean exists in the User model but is never visibly enforced in the UI. New recruits should be forced to set a personal password on first login.

### Medium-term

5. **Comparative group analytics**  
   Histogram / percentile distribution of test scores per group or division. Recharts is already in the dependency tree.

6. **Custom report builder**  
   Select user + tests → generate a structured PDF report with scores, interpretations, and examiner notes. Use `pdfkit` or `puppeteer` in the Electron main process.

7. **Excel import for user roster**  
   Bulk-create users from an Excel spreadsheet of recruits. `xlsx` is already installed.

8. **Role-based feature gating**  
   Currently `role` is a free-text string. Implement proper RBAC: `admin` can manage users and upload tests; `psychologist` can assign and score tests; `user` (subject) can only take tests.

### Long-term

9. **Encrypted SQLite backup**  
   Scheduled export of the SQLite file, encrypted with AES-256, written to a designated backup path. Important for data continuity.

10. **Network mode (optional)**  
    Allow the app to run as a server that multiple clients connect to simultaneously, without requiring individual Electron instances. This would require moving from SQLite to PostgreSQL.

---

## Part 5 — UI/UX Design Extensions

### Information Architecture

The current structure is flat and functional but lacks workflow orientation. Users (assessors) follow a repeating cycle:
`Recruit arrives → Create profile → Assign test battery → Review scores → Write summary → File report`

The dashboard should surface this workflow explicitly rather than exposing raw data tables.

**Proposed navigation redesign:**

```
Призма
├── Панель управления       (KPIs: today's sessions, pending reviews, score outliers)
├── Личный состав           (user roster with inline status: tested / pending / summarized)
│   └── [userId]            (full profile + all submissions + summary history)
├── Методики                (test & form library)
│   └── [testId]            (preview + score key + normative tables)
├── Сессии                  (active and completed assessment sessions)
├── Аналитика               (group stats, score distributions)
├── Архив                   (closed cases)
└── Настройки               (roles, backup, app preferences)
```

---

### Key UI Improvements

**1. Personnel profile page**  
Currently just raw fields. Redesign as a structured card:
- Header: full name + rank badge + division + avatar/initials
- Tabs: "Профиль" / "Тесты" / "Бланки" / "Заключение"
- Test history timeline with pass/fail indicators and score sparklines
- "Назначить тест" CTA button prominently placed

**2. Test result display**  
After a submission, show:
- Raw scores per scale
- Converted stan/T-grade scores with color coding (green/yellow/red zones)
- Interpretation text from `summaryTable`
- Radar chart (Recharts `RadarChart`) for multi-scale profiles
- "Сформировать заключение" button that pre-fills the `UserSummary` verdict

**3. Test-taking UX**  
- Full-screen mode for focus during assessment
- Visible progress indicator (question N of M)
- Countdown timer when `ttc` is set
- Auto-save state in case of accidental close (use Zustand + localStorage)
- Review screen before final submit

**4. Dashboard KPIs**  
Replace the generic statistics panel with actionable widgets:
- "Ожидают заключения" (submissions with empty summary) — click-through to those users
- "Тесты сегодня" — count of sessions completed today
- "Группы без тестирования" — groups with no tests in the last N days
- Score outlier alert (users scoring in critical zones)

**5. Data table improvements**  
The `data-table.tsx` component (TanStack Table) should consistently support:
- Column visibility toggle
- Sort by any column
- Text search
- Export selected rows to CSV/Excel

**6. Theme and visual identity**  
- `theme-toggle.tsx` exists but the dark theme needs a full design pass
- Use a professional, calm color palette suited to a clinical/institutional context (avoid playful or aggressive colors)
- Add the app name "Призма" as a wordmark in the sidebar
- Minimum window size: 1280×768 with responsive column hiding below that

**7. Empty states**  
Every data table and list needs a designed empty state:
- No users yet → "Добавьте первого пользователя"
- No tests → "Загрузите методики в разделе Методики"
- No submissions → "Назначьте тест пользователю"

**8. Onboarding flow**  
On first launch (admin creates account), show a 3-step wizard:
1. Create admin account
2. Upload first test methodology
3. Add first recruit
This replaces the current blank dashboard that new users see.

---

## Priority Matrix

| Priority | Item |
|----------|------|
| **P0 — Ship blocker** | Remove `.env` and `prisma.exe` from git; fix `signUp(any)`; fix missing relations |
| **P1 — Correctness** | Rename `useSession`; fix non-null assertion; enforce `firstTimer` flow; add Zod to all server actions |
| **P2 — Robustness** | Pagination; loading.tsx + error.tsx; Electron wait-on; enums for role/rank/servingKind |
| **P3 — UX** | Test result display with scores + charts; personnel profile redesign; empty states; timed tests |
| **P4 — Features** | PDF export; bulk test assignment; group analytics; Excel roster import; audit log |
| **P5 — Scale** | Encrypted backup; network mode; full RBAC |
