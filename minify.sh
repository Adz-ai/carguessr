#!/bin/bash

# Create minified directory if it doesn't exist
mkdir -p static/js
mkdir -p static/css

echo "Minifying JavaScript files using npx..."
# Minify JavaScript files using npx (no global install needed)
npx uglify-js static/js/game.js -c -m -o static/js/game.min.js
npx uglify-js static/js/auth.js -c -m -o static/js/auth.min.js

echo "Minifying CSS files using npx..."
# Minify CSS files using npx (no global install needed)
npx clean-css-cli -o static/css/styles.min.css static/css/styles.css

echo "Minification complete!"
echo ""
echo "Original sizes:"
ls -lh static/js/game.js static/js/auth.js static/css/styles.css
echo ""
echo "Minified sizes:"
ls -lh static/js/game.min.js static/js/auth.min.js static/css/styles.min.css