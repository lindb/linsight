.PHONY: help build test deps generate clean

web_build: web_clean
	yarn install
	yarn build

web_clean:
	rm -rf static/assets
	find static ! -name '.gitignore' ! -name 'README.md'  -type f -exec rm -rf {} +

web_clean_all:
	rm -rf node_modules
	rm -rf static/assets
	find static ! -name '.gitignore' ! -name 'README.md'  -type f -exec rm -rf {} +
