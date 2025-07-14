package database

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type JSONDB struct {
	filename string
	data     map[string][]Entry     // {"exact": [], "contains": [], "regex": []}
	models   map[string][]ModelInfo // {"openai": [], "anthropic": [], ...}
}

func NewJSONDB(filename string) (*JSONDB, error) {
	db := &JSONDB{filename: filename}
	if err := db.Reload(); err != nil {
		return nil, err
	}
	return db, nil
}

// Implement the combined functions
func (j *JSONDB) Query(query string) ([]Entry, error) {
	var allEntries []Entry

	exactEntries, err := j.QueryExact(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, exactEntries...)

	containsEntries, err := j.QueryContains(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, containsEntries...)

	regexEntries, err := j.QueryRegex(query)
	if err != nil {
		return nil, err
	}
	allEntries = append(allEntries, regexEntries...)

	return allEntries, nil
}

func (j *JSONDB) QueryByID(id int, matchType int) (*Entry, error) {
	var tableName string
	switch matchType {
	case 1:
		tableName = "exact"
	case 2:
		tableName = "contains"
	case 3:
		tableName = "regex"
	default:
		return nil, fmt.Errorf("invalid match type: %d", matchType)
	}

	entries, ok := j.data[tableName]
	if !ok {
		return nil, fmt.Errorf("no entries found for match type: %s", tableName)
	}

	for _, entry := range entries {
		if entry.ID == id {
			entry.MatchType = matchType // Set the MatchType before returning
			return &entry, nil
		}
	}

	return nil, fmt.Errorf("entry with ID %d not found in %s", id, tableName)
}

func (j *JSONDB) AddEntry(key string, matchType int, value string) error {
	switch matchType {
	case 1:
		return j.AddEntryExact(key, value)
	case 2:
		return j.AddEntryContains(key, value)
	case 3:
		return j.AddEntryRegex(key, value)
	default:
		return fmt.Errorf("invalid match type: %d", matchType)
	}
}

func (j *JSONDB) UpdateEntry(key string, oldType int, newType int, value string) error {
	if oldType == newType {
		// Same type, use existing UpdateEntryXXX functions
		switch oldType {
		case 1:
			return j.UpdateEntryExact(key, value)
		case 2:
			return j.UpdateEntryContains(key, value)
		case 3:
			return j.UpdateEntryRegex(key, value)
		default:
			return fmt.Errorf("invalid match type: %d", oldType)
		}
	} else {
		// Different types, delete from old and add to new
		if err := j.DeleteEntry(key, oldType); err != nil {
			return err
		}
		return j.AddEntry(key, newType, value)
	}
}

func (j *JSONDB) DeleteEntry(key string, matchType int) error {
	switch matchType {
	case 1:
		return j.DeleteEntryExact(key)
	case 2:
		return j.DeleteEntryContains(key)
	case 3:
		return j.DeleteEntryRegex(key)
	default:
		return fmt.Errorf("invalid match type: %d", matchType)
	}
}

func (j *JSONDB) ListEntries(table string) ([]Entry, error) {
	var allEntries []Entry
	for _, matchType := range table {
		var entries []Entry
		var err error

		switch matchType {
		case 1:
			entries, err = j.listEntries("exact")
			if err == nil {
				for i := range entries {
					entries[i].MatchType = 1
				}
			}
		case 2:
			entries, err = j.listEntries("contains")
			if err == nil {
				for i := range entries {
					entries[i].MatchType = 2
				}
			}
		case 3:
			entries, err = j.listEntries("regex")
			if err == nil {
				for i := range entries {
					entries[i].MatchType = 3
				}
			}
		default:
			return nil, fmt.Errorf("invalid match type: %d", matchType)
		}

		if err != nil {
			return nil, err
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

func (j *JSONDB) ListSpecificEntries(matchTypes ...int) ([]Entry, error) {
	if len(matchTypes) == 0 {
		// List all entries if no match types are specified
		return j.ListAllEntries()
	}

	var allEntries []Entry
	for _, matchType := range matchTypes {
		var entries []Entry
		var err error

		switch matchType {
		case 1:
			entries, err = j.ListEntriesExact()
		case 2:
			entries, err = j.ListEntriesContains()
		case 3:
			entries, err = j.ListEntriesRegex()
		default:
			return nil, fmt.Errorf("invalid match type: %d", matchType)
		}

		if err != nil {
			return nil, err
		}
		for i := range entries {
			entries[i].MatchType = matchType
		}
		allEntries = append(allEntries, entries...)
	}

	return allEntries, nil
}

func (j *JSONDB) ListAllEntries() ([]Entry, error) {
	var allEntries []Entry
	for matchType, entries := range j.data {
		for i := range entries {
			switch matchType {
			case "exact":
				entries[i].MatchType = 1
			case "contains":
				entries[i].MatchType = 2
			case "regex":
				entries[i].MatchType = 3
			}
		}
		allEntries = append(allEntries, entries...)
	}
	return allEntries, nil
}

func (j *JSONDB) QueryExact(query string) ([]Entry, error) {
	return j.query(query, "exact")
}

func (j *JSONDB) QueryContains(query string) ([]Entry, error) {
	return j.query(query, "contains")
}

func (j *JSONDB) QueryRegex(query string) ([]Entry, error) {
	return j.query(query, "regex")
}

func (j *JSONDB) query(query string, matchType string) ([]Entry, error) {
	var results []Entry
	entries, ok := j.data[matchType]
	if !ok {
		return []Entry{}, nil
	}
	switch matchType {
	case "exact":
		for _, entry := range entries {
			if entry.Key == query {
				entry.MatchType = 1
				results = append(results, entry)
			}
		}
	case "contains":
		for _, entry := range entries {
			if strings.Contains(query, entry.Key) {
				entry.MatchType = 2
				results = append(results, entry)
			}
		}
	case "regex":
		for _, entry := range entries {
			matched, _ := regexp.MatchString(entry.Key, query)
			if matched {
				entry.MatchType = 3
				results = append(results, entry)
			}
		}
	}
	return results, nil
}

func (j *JSONDB) AddEntryExact(key string, value string) error {
	return j.addEntry(key, value, "exact")
}

func (j *JSONDB) AddEntryContains(key string, value string) error {
	return j.addEntry(key, value, "contains")
}

func (j *JSONDB) AddEntryRegex(key string, value string) error {
	return j.addEntry(key, value, "regex")
}

func (j *JSONDB) addEntry(key string, value string, matchType string) error {
	entries := j.data[matchType]
	newID := len(entries) + 1
	idExists := func(id int) bool {
		for _, entry := range entries {
			if entry.ID == id {
				return true
			}
		}
		return false
	}

	for idExists(newID) {
		newID++
	}

	var matchTypeInt int
	switch matchType {
	case "exact":
		matchTypeInt = 1
	case "contains":
		matchTypeInt = 2
	case "regex":
		matchTypeInt = 3
	default:
		return fmt.Errorf("invalid match type: %s", matchType)
	}

	newEntry := Entry{
		ID:        newID,
		Key:       key,
		Value:     value,
		MatchType: matchTypeInt,
	}
	j.data[matchType] = append(entries, newEntry)
	return j.Save()
}

func (j *JSONDB) UpdateEntryExact(key string, value string) error {
	return j.updateEntry(key, value, "exact")
}

func (j *JSONDB) UpdateEntryContains(key string, value string) error {
	return j.updateEntry(key, value, "contains")
}

func (j *JSONDB) UpdateEntryRegex(key string, value string) error {
	return j.updateEntry(key, value, "regex")
}

func (j *JSONDB) updateEntry(key string, value string, matchType string) error {
	entries := j.data[matchType]
	for i, entry := range entries {
		if entry.Key == key {
			j.data[matchType][i].Value = value
			switch matchType {
			case "exact":
				j.data[matchType][i].MatchType = 1
			case "contains":
				j.data[matchType][i].MatchType = 2
			case "regex":
				j.data[matchType][i].MatchType = 3
			}
			return j.Save()
		}
	}
	return fmt.Errorf("entry not found")
}

func (j *JSONDB) DeleteEntryExact(key string) error {
	return j.deleteEntry(key, "exact")
}

func (j *JSONDB) DeleteEntryContains(key string) error {
	return j.deleteEntry(key, "contains")
}

func (j *JSONDB) DeleteEntryRegex(key string) error {
	return j.deleteEntry(key, "regex")
}

func (j *JSONDB) deleteEntry(key string, matchType string) error {
	entries := j.data[matchType]
	for i, entry := range entries {
		if entry.Key == key {
			j.data[matchType] = append(entries[:i], entries[i+1:]...)
			return j.Save()
		}
	}
	return fmt.Errorf("entry not found")
}

func (j *JSONDB) ListEntriesExact() ([]Entry, error) {
	return j.listEntries("exact")
}

func (j *JSONDB) ListEntriesContains() ([]Entry, error) {
	return j.listEntries("contains")
}

func (j *JSONDB) ListEntriesRegex() ([]Entry, error) {
	return j.listEntries("regex")
}

func (j *JSONDB) listEntries(matchType string) ([]Entry, error) {
	entries, ok := j.data[matchType]
	if !ok {
		return []Entry{}, nil
	}
	return entries, nil
}

func (j *JSONDB) Reload() error {
	file, err := os.Open(j.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if len(bytes) == 0 {
		// Initialize with empty data if the file is empty
		j.data = map[string][]Entry{
			"exact":    {},
			"contains": {},
			"regex":    {},
		}
		j.models = make(map[string][]ModelInfo)
		return nil
	}

	// 尝试解析新的格式（包含models）
	var fullData map[string]interface{}
	err = json.Unmarshal(bytes, &fullData)
	if err != nil {
		return err
	}

	// 初始化数据结构
	j.data = make(map[string][]Entry)
	j.models = make(map[string][]ModelInfo)

	// 解析FAQ数据
	for key, value := range fullData {
		if key == "models" {
			// 解析模型数据
			if modelsData, ok := value.(map[string]interface{}); ok {
				for provider, models := range modelsData {
					if modelList, ok := models.([]interface{}); ok {
						var modelInfos []ModelInfo
						for _, model := range modelList {
							if modelMap, ok := model.(map[string]interface{}); ok {
								modelInfo := ModelInfo{
									ID:          getString(modelMap, "id"),
									Name:        getString(modelMap, "name"),
									Provider:    getString(modelMap, "provider"),
									Description: getString(modelMap, "description"),
									UpdatedAt:   getString(modelMap, "updated_at"),
								}
								modelInfos = append(modelInfos, modelInfo)
							}
						}
						j.models[provider] = modelInfos
					}
				}
			}
		} else {
			// 解析FAQ条目数据
			if entryList, ok := value.([]interface{}); ok {
				var entries []Entry
				for _, entry := range entryList {
					if entryMap, ok := entry.(map[string]interface{}); ok {
						entryInfo := Entry{
							ID:        int(getFloat64(entryMap, "id")),
							Key:       getString(entryMap, "key"),
							Value:     getString(entryMap, "value"),
							MatchType: int(getFloat64(entryMap, "match_type")),
						}
						entries = append(entries, entryInfo)
					}
				}
				j.data[key] = entries
			}
		}
	}

	// Ensure all match types exist in the data
	if _, ok := j.data["exact"]; !ok {
		j.data["exact"] = []Entry{}
	}
	if _, ok := j.data["contains"]; !ok {
		j.data["contains"] = []Entry{}
	}
	if _, ok := j.data["regex"]; !ok {
		j.data["regex"] = []Entry{}
	}

	return nil
}

func (j *JSONDB) DeleteAllEntries() error {
	j.data["exact"] = []Entry{}
	j.data["contains"] = []Entry{}
	j.data["regex"] = []Entry{}
	return j.Save()
}

func (j *JSONDB) Save() error {
	bytes, err := json.MarshalIndent(j.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(j.filename, bytes, 0644)
}

func (j *JSONDB) Close() error {
	// No resources to close for JSONDB
	return nil
}

// 模型管理功能
func (j *JSONDB) SaveModels(provider string, models []ModelInfo) error {
	if j.models == nil {
		j.models = make(map[string][]ModelInfo)
	}
	j.models[provider] = models
	return j.SaveWithModels()
}

func (j *JSONDB) GetModels(provider string) ([]ModelInfo, error) {
	if j.models == nil {
		return []ModelInfo{}, nil
	}
	return j.models[provider], nil
}

func (j *JSONDB) GetAllModels() (map[string][]ModelInfo, error) {
	if j.models == nil {
		return make(map[string][]ModelInfo), nil
	}
	return j.models, nil
}

func (j *JSONDB) DeleteModels(provider string) error {
	if j.models == nil {
		return nil
	}
	delete(j.models, provider)
	return j.SaveWithModels()
}

func (j *JSONDB) SaveWithModels() error {
	// 创建包含FAQ数据和模型数据的完整结构
	fullData := map[string]interface{}{
		"exact":    j.data["exact"],
		"contains": j.data["contains"],
		"regex":    j.data["regex"],
		"models":   j.models,
	}

	bytes, err := json.MarshalIndent(fullData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(j.filename, bytes, 0644)
}

// 辅助函数用于类型转换
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloat64(m map[string]interface{}, key string) float64 {
	if val, ok := m[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0
}
