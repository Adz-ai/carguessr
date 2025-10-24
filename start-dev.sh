#!/bin/bash

# CarGuessr Development Start Script

echo "🚀 Starting CarGuessr in Development Mode"
echo ""
echo "This will start:"
echo "  1. Go backend on http://localhost:8080"
echo "  2. React frontend on http://localhost:5173"
echo ""
echo "Press Ctrl+C to stop both servers"
echo ""

# Function to cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping servers..."
    kill 0
    exit
}

trap cleanup INT TERM

# Start Go server in background
echo "📦 Starting Go backend..."
cd "$(dirname "$0")"
go run cmd/server/main.go &
BACKEND_PID=$!

# Wait for backend to start
sleep 2

# Start React dev server
echo "⚛️  Starting React frontend..."
cd frontend
npm run dev &
FRONTEND_PID=$!

# Wait for both processes
wait
