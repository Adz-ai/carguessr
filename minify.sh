#!/bin/bash

# Create minified directory if it doesn't exist
mkdir -p static/js/min
mkdir -p static/css/min

echo "Installing minification tools..."
# Check if uglify-js is installed
if ! command -v uglifyjs &> /dev/null; then
    echo "Installing uglify-js..."
    npm install -g uglify-js
fi

# Check if clean-css-cli is installed
if ! command -v cleancss &> /dev/null; then
    echo "Installing clean-css-cli..."
    npm install -g clean-css-cli
fi

echo "Minifying JavaScript files..."
# Minify JavaScript files
uglifyjs static/js/game.js -c -m -o static/js/game.min.js
uglifyjs static/js/auth.js -c -m -o static/js/auth.min.js

echo "Minifying CSS files..."
# Minify CSS files
cleancss -o static/css/styles.min.css static/css/styles.css

echo "Minification complete!"
echo ""
echo "Original sizes:"
ls -lh static/js/game.js static/js/auth.js static/css/styles.css
echo ""
echo "Minified sizes:"
ls -lh static/js/game.min.js static/js/auth.min.js static/css/styles.min.css