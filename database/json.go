package database

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"TGFaqBot/config"
)

type JSONDB struct {
	filename   string
	data       map[string][]Entry     // {"exact": [], "contains": [], "regex": []}
	models     map[string][]ModelInfo // {"openai": [], "anthropic": [], ...}
	modelCache []config.Model         // 缓存的模型列表
	cacheTime  string                 // 缓存时间
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

func (j *JSONDB) QueryByID(id int, matchType MatchType) (*Entry, error) {
	tableName := matchType.GetTableName()

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

func (j *JSONDB) AddEntry(key string, matchType MatchType, value string) error {
	switch matchType {
	case MatchExact:
		return j.AddEntryExact(key, value)
	case MatchContains:
		return j.AddEntryContains(key, value)
	case MatchRegex:
		return j.AddEntryRegex(key, value)
	default:
		return fmt.Errorf("invalid match type: %s", matchType)
	}
}

func (j *JSONDB) UpdateEntry(key string, oldType MatchType, newType MatchType, value string) error {
	if oldType == newType {
		// Same type, use existing UpdateEntryXXX functions
		switch oldType {
		case MatchExact:
			return j.UpdateEntryExact(key, value)
		case MatchContains:
			return j.UpdateEntryContains(key, value)
		case MatchRegex:
			return j.UpdateEntryRegex(key, value)
		default:
			return fmt.Errorf("invalid match type: %s", oldType)
		}
	} else {
		// Different types, delete from old and add to new
		if err := j.DeleteEntry(key, oldType); err != nil {
			return err
		}
		return j.AddEntry(key, newType, value)
	}
}

func (j *JSONDB) DeleteEntry(key string, matchType MatchType) error {
	switch matchType {
	case MatchExact:
		return j.DeleteEntryExact(key)
	case MatchContains:
		return j.DeleteEntryContains(key)
	case MatchRegex:
		return j.DeleteEntryRegex(key)
	default:
		return fmt.Errorf("invalid match type: %s", matchType)
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
					entries[i].MatchType = MatchExact
				}
			}
		case 2:
			entries, err = j.listEntries("contains")
			if err == nil {
				for i := range entries {
					entries[i].MatchType = MatchContains
				}
			}
		case 3:
			entries, err = j.listEntries("regex")
			if err == nil {
				for i := range entries {
					entries[i].MatchType = MatchRegex
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

func (j *JSONDB) ListSpecificEntries(matchTypes ...MatchType) ([]Entry, error) {
	if len(matchTypes) == 0 {
		// List all entries if no match types are specified
		return j.ListAllEntries()
	}

	var allEntries []Entry
	for _, matchType := range matchTypes {
		var entries []Entry
		var err error

		switch matchType {
		case MatchExact:
			entries, err = j.ListEntriesExact()
		case MatchContains:
			entries, err = j.ListEntriesContains()
		case MatchRegex:
			entries, err = j.ListEntriesRegex()
		default:
			return nil, fmt.Errorf("invalid match type: %s", matchType)
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
				entries[i].MatchType = MatchExact
			case "contains":
				entries[i].MatchType = MatchContains
			case "regex":
				entries[i].MatchType = MatchRegex
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
				entry.MatchType = MatchExact
				results = append(results, entry)
			}
		}
	case "contains":
		for _, entry := range entries {
			if strings.Contains(query, entry.Key) {
				entry.MatchType = MatchContains
				results = append(results, entry)
			}
		}
	case "regex":
		for _, entry := range entries {
			matched, _ := regexp.MatchString(entry.Key, query)
			if matched {
				entry.MatchType = MatchRegex
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
		MatchType: intToMatchType(matchTypeInt),
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
				j.data[matchType][i].MatchType = MatchExact
			case "contains":
				j.data[matchType][i].MatchType = MatchContains
			case "regex":
				j.data[matchType][i].MatchType = MatchRegex
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
		j.modelCache = []config.Model{}
		j.cacheTime = ""
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
	j.modelCache = []config.Model{}
	j.cacheTime = ""

	// 解析FAQ数据
	for key, value := range fullData {
		switch key {
		case "models":
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
		case "model_cache":
			// 解析模型缓存数据
			if cacheData, ok := value.(map[string]interface{}); ok {
				if cacheTime, ok := cacheData["cache_time"].(string); ok {
					j.cacheTime = cacheTime
				}
				if modelList, ok := cacheData["models"].([]interface{}); ok {
					for _, model := range modelList {
						if modelMap, ok := model.(map[string]interface{}); ok {
							modelInfo := config.Model{
								ID:       getString(modelMap, "id"),
								Name:     getString(modelMap, "name"),
								Provider: getString(modelMap, "provider"),
							}
							j.modelCache = append(j.modelCache, modelInfo)
						}
					}
				}
			}
		default:
			// 解析FAQ条目数据
			if entryList, ok := value.([]interface{}); ok {
				var entries []Entry
				for _, entry := range entryList {
					if entryMap, ok := entry.(map[string]interface{}); ok {
						entryInfo := Entry{
							ID:        int(getFloat64(entryMap, "id")),
							Key:       getString(entryMap, "key"),
							Value:     getString(entryMap, "value"),
							MatchType: intToMatchType(int(getFloat64(entryMap, "match_type"))),
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
	// 创建包含FAQ数据、模型数据和缓存数据的完整结构
	fullData := map[string]interface{}{
		"exact":    j.data["exact"],
		"contains": j.data["contains"],
		"regex":    j.data["regex"],
		"models":   j.models,
	}

	// 添加模型缓存数据
	if len(j.modelCache) > 0 {
		fullData["model_cache"] = map[string]interface{}{
			"models":     j.modelCache,
			"cache_time": j.cacheTime,
		}
	}

	bytes, err := json.MarshalIndent(fullData, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(j.filename, bytes, 0644)
}

// 模型缓存接口实现
func (j *JSONDB) SetModelCache(models []config.Model, updatedAt string) error {
	j.modelCache = models
	j.cacheTime = updatedAt
	return j.SaveWithModels()
}

func (j *JSONDB) GetModelCache() ([]config.Model, string, error) {
	return j.modelCache, j.cacheTime, nil
}

func (j *JSONDB) ClearModelCache() error {
	j.modelCache = []config.Model{}
	j.cacheTime = ""
	return j.SaveWithModels()
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

// Telegraph 内容管理方法
func (j *JSONDB) AddTelegraphEntry(key string, matchType MatchType, value string, contentType string, telegraphURL string, telegraphPath string) error {
	// 暂时简化实现，将 Telegraph URL 存储在 value 字段中
	return j.AddEntry(key, matchType, telegraphURL)
}

func (j *JSONDB) UpdateTelegraphEntry(key string, matchType MatchType, value string, contentType string, telegraphURL string, telegraphPath string) error {
	// 暂时简化实现
	return j.UpdateEntry(key, matchType, matchType, telegraphURL)
}

func (j *JSONDB) GetTelegraphContent(key string, matchType MatchType) (*Entry, error) {
	// 暂时使用现有的查询方法
	return j.QueryByID(1, matchType) // 默认ID为1
}
