# User Feedback & Implementation Status

## Mobile/UX Issues ✅ FIXED
- ✅ **The CSS on phones might need touching up cause I can scroll left to right on iPhone 13 pro** - Fixed horizontal scroll issues on mobile
- ✅ **It's fun, one recommendation is on mobile maybe make the main image aspect ratio 16:9 or the original aspect ratio as the square aspect ratio kinda cuts some images off** - Changed to 4:3 aspect ratio to prevent image cropping
- ✅ **Leaderboard modal fixes** - Fixed close button accessibility, prevented background scrolling, improved mobile layout

## Game Logic Issues ✅ FIXED  
- ✅ **Only issue is within 4-5 guesses it rotated in the first car again, aside from that cool game** - Implemented session-based car history tracking to prevent repetition (tracks last 10 cars per session)

## Feature Requests 🔮 FUTURE
- 🔮 **Is he gonna turn it into an app? Coz that would probs be better** - Consider PWA or native app development
- 🔮 **Also should have option to create an account or maybe play against a friend** - User accounts and multiplayer features
- 🔮 **Love this thank you properly good fun. Would be fun to have a way to challenge friends with a code for a specific set of cars like you can in geoguessr** - Friend challenge codes/shared challenges
- 🔮 **Had a mess about and wow I am garbage hahaha, It's a fun wee game, nice work. Only thing I'd say is the ui on the first page, I'd make it a 2 step process** - UI simplification (difficulty first, then game mode)

## Additional Fixes Implemented ✅
- ✅ **Challenge mode button text** - Fixed "View on Bonhams" button to show correct source (Lookers vs Bonhams) based on difficulty
- ✅ **Game mode badges** - Added "Endless", "Difficult", and "Recommended" badges for better UX
- ✅ **GitHub attribution** - Added footer with creator credit
- ✅ **Europe warning** - Added warning for Easy mode image issues outside Europe
- ✅ **Mobile auto-scroll** - Added auto-scroll to car image after price submission on mobile for better UX
