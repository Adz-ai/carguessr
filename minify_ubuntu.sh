#!/bin/bash

echo "Creating minified files for Ubuntu deployment..."

# For Ubuntu, we'll use simple alternatives or create pre-minified files

# Option 1: Use terser (easier to install on Ubuntu)
if command -v terser &> /dev/null; then
    echo "Using terser for JavaScript minification..."
    terser static/js/game.js -c -m -o static/js/game.min.js
    terser static/js/auth.js -c -m -o static/js/auth.min.js
else
    echo "Terser not found. Install with: sudo apt-get install node-terser"
    echo "Or use pre-minified files from development machine"
fi

# Option 2: For CSS, we can use a simple sed command to remove comments and whitespace
echo "Minifying CSS..."
# Remove comments and unnecessary whitespace
sed 's/\/\*[^*]*\*\///g; s/  */ /g; s/: /:/g; s/; /;/g; s/ {/{/g; s/{ /{/g; s/ }/}/g' static/css/styles.css > static/css/styles.min.css

echo "Minification complete!"
echo ""
echo "File sizes:"
ls -lh static/js/game.js static/js/game.min.js 2>/dev/null
ls -lh static/js/auth.js static/js/auth.min.js 2>/dev/null  
ls -lh static/css/styles.css static/css/styles.min.css 2>/dev/null

echo ""
echo "Note: For best results, minify on your development machine and copy the .min files"