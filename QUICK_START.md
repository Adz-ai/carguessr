# CarGuessr - Quick Start Guide

## 🎉 Migration Complete!

Your CarGuessr app has been fully migrated to React + TypeScript + Vite.

---

## 🚀 Start Developing (Choose One)

### Option 1: Automated Start (Recommended)
```bash
./start-dev.sh
```
This starts both servers automatically.

### Option 2: Manual Start

**Terminal 1 - Backend:**
```bash
go run cmd/server/main.go
```

**Terminal 2 - Frontend:**
```bash
cd frontend
npm run dev
```

Then visit: **http://localhost:5173**

---

## 📦 Production Build

```bash
cd frontend
npm run build
cd ..
go run cmd/server/main.go
```

Then visit: **http://localhost:8080**

---

## 📁 Key Files

| File | Purpose |
|------|---------|
| `frontend/src/App.tsx` | Main app component |
| `frontend/src/components/` | All React components |
| `frontend/src/stores/` | State management |
| `frontend/src/api/client.ts` | API calls |
| `frontend/src/types/index.ts` | TypeScript types |
| `cmd/server/main.go` | Go backend server |

---

## 🎮 Features

✅ All 3 game modes (Challenge, Streak, Stay at Zero)
✅ Easy & Hard difficulty
✅ Full authentication system
✅ Leaderboards
✅ Friend challenges
✅ User profiles
✅ Mobile responsive

---

## 📚 Documentation

- **Deployment Guide**: `frontend/DEPLOYMENT.md`
- **Migration Complete**: `MIGRATION_COMPLETE.md`
- **Technical Details**: `frontend/SUMMARY.md`

---

## 🐛 Troubleshooting

### Build fails?
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
npm run build
```

### CORS errors?
- Ensure Go server is running on port 8080
- Check both servers are running

### Hot reload not working?
```bash
cd frontend
npm run dev
```

---

## ✨ You're All Set!

Everything is working and ready to use. Happy coding! 🎉
