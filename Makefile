# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GORUN=$(GOCMD) run

# Binary names
COMPRESSOR_BINARY=ppma_compress
DECOMPRESSOR_BINARY=ppma_decompress
CALGARY_BINARY=calgary_analysis

# Paths to main packages
COMPRESSOR_MAIN=./cmd/compressor
DECOMPRESSOR_MAIN=./cmd/decompressor
CALGARY_MAIN=./cmd/calgary_analysis

# Output directory
BIN_DIR=./bin
REPORTS_DIR=./test/reports

# Test dataset dir
DIR=./test/test_dataset

# Dataset URL and zip file
DATASET_URL=https://eugeniy-belyaev.narod.ru/InfTheory/CalgaryCorpus.zip
DATASET_ZIP=$(DIR)/CalgaryCorpus.zip

all: build

build: clean $(BIN_DIR) $(BIN_DIR)/$(COMPRESSOR_BINARY) $(BIN_DIR)/$(DECOMPRESSOR_BINARY) $(BIN_DIR)/$(CALGARY_BINARY)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(REPORTS_DIR):
	mkdir -p $(REPORTS_DIR)

$(DIR):
	mkdir -p $(DIR)

# Скачивание и распаковка датасета
download-dataset: $(DIR)
	@echo "Downloading Calgary Corpus..."
	@if command -v wget >/dev/null 2>&1; then \
		wget -O $(DATASET_ZIP) $(DATASET_URL); \
	elif command -v curl >/dev/null 2>&1; then \
		curl -L -o $(DATASET_ZIP) $(DATASET_URL); \
	else \
		echo "Error: Neither wget nor curl found. Please install one of them."; \
		exit 1; \
	fi
	@echo "Extracting..."
	@unzip -q $(DATASET_ZIP) -d $(DIR)
	@echo "Removing zip..."
	@rm $(DATASET_ZIP)
	@echo "Dataset ready in $(DIR)"

# Зависимость для целей, которым нужен датасет
ensure-dataset: $(DIR)
	@if [ -z "$$(ls -A $(DIR) 2>/dev/null)" ]; then \
		$(MAKE) download-dataset; \
	fi

$(BIN_DIR)/$(COMPRESSOR_BINARY): $(COMPRESSOR_MAIN)
	$(GOBUILD) -o $(BIN_DIR)/$(COMPRESSOR_BINARY) $(COMPRESSOR_MAIN)

$(BIN_DIR)/$(DECOMPRESSOR_BINARY): $(DECOMPRESSOR_MAIN)
	$(GOBUILD) -o $(BIN_DIR)/$(DECOMPRESSOR_BINARY) $(DECOMPRESSOR_MAIN)

$(BIN_DIR)/$(CALGARY_BINARY): $(CALGARY_MAIN)
	$(GOBUILD) -o $(BIN_DIR)/$(CALGARY_BINARY) $(CALGARY_MAIN)

clean:
	rm -rf $(BIN_DIR) $(REPORTS_DIR)

clean-dataset:
	rm -rf $(DIR)

test:
	$(GOTEST) ./...

test-e2e: build
	$(GORUN) ./cmd/e2e

# Запуск анализа Calgary corpus (автоматически скачивает датасет, если его нет)
calgary: ensure-dataset $(BIN_DIR)/$(CALGARY_BINARY)
	$(BIN_DIR)/$(CALGARY_BINARY) -dir=$(DIR)

# Запуск анализа с сохранением результатов в файл (текстовая таблица)
calgary-save: ensure-dataset $(REPORTS_DIR) $(BIN_DIR)/$(CALGARY_BINARY)
	$(BIN_DIR)/$(CALGARY_BINARY) -dir=$(DIR) -output=$(REPORTS_DIR)/calgary_$(shell date +%Y%m%d_%H%M%S).txt

# Запуск без сборки (удобно для разработки) – тоже с автоматической загрузкой
run-calgary: ensure-dataset
	$(GORUN) $(CALGARY_MAIN) -dir=$(DIR)

run-calgary-save: ensure-dataset $(REPORTS_DIR)
	$(GORUN) $(CALGARY_MAIN) -dir=$(DIR) -output=$(REPORTS_DIR)/calgary_$(shell date +%Y%m%d_%H%M%S).txt

.PHONY: all build clean clean-dataset test test-e2e calgary calgary-save run-calgary run-calgary-save download-dataset ensure-dataset