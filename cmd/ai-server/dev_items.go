package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ai-server/internal/session"
)

type devItemRole struct {
	PlayerID    string                 `json:"playerId"`
	RoleID      string                 `json:"roleId"`
	DisplayName string                 `json:"displayName"`
	Level       int                    `json:"level"`
	Exp         int                    `json:"exp"`
	Vocation    string                 `json:"vocation"`
	Currencies  session.RoleCurrencies `json:"currencies"`
	Items       []session.RoleItem     `json:"items"`
	Capacity    int                    `json:"capacity"`
}

type devItemTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Display     string `json:"display"`
	ItemType    string `json:"itemType"`
	Description string `json:"description"`
}

type devItemsStateResponse struct {
	Roles     []devItemRole     `json:"roles"`
	Templates []devItemTemplate `json:"templates"`
}

type devAddItemRequest struct {
	PlayerID string `json:"playerId"`
	RoleID   string `json:"roleId"`
	ItemID   string `json:"itemId"`
	Count    int    `json:"count"`
}

type devAddItemResponse struct {
	Success      bool             `json:"success"`
	ErrorCode    string           `json:"errorCode,omitempty"`
	ErrorMessage string           `json:"errorMessage,omitempty"`
	Role         *devItemRole     `json:"role,omitempty"`
	Item         session.RoleItem `json:"item,omitempty"`
}

type devAddCurrencyRequest struct {
	PlayerID string `json:"playerId"`
	RoleID   string `json:"roleId"`
	Amount   int    `json:"amount"`
}

type devAddCurrencyResponse struct {
	Success      bool                   `json:"success"`
	ErrorCode    string                 `json:"errorCode,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Role         *devItemRole           `json:"role,omitempty"`
	Currencies   session.RoleCurrencies `json:"currencies,omitempty"`
}

type devSetLevelRequest struct {
	PlayerID string `json:"playerId"`
	RoleID   string `json:"roleId"`
	Level    int    `json:"level"`
}

type devSetLevelResponse struct {
	Success      bool         `json:"success"`
	ErrorCode    string       `json:"errorCode,omitempty"`
	ErrorMessage string       `json:"errorMessage,omitempty"`
	Role         *devItemRole `json:"role,omitempty"`
}

func registerDevItemHandlers(mux *http.ServeMux, store *session.Store) {
	mux.HandleFunc("/dev/items", devItemsPageHandler)
	mux.Handle("/dev/classic-icons/", http.StripPrefix("/dev/classic-icons/", http.FileServer(http.Dir(resolveDevClassicIconDir()))))
	mux.HandleFunc("/dev/items/state", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeDevJSON(writer, buildDevItemsState(store))
	})
	mux.HandleFunc("/dev/items/add", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleDevAddItem(writer, request, store)
	})
	mux.HandleFunc("/dev/items/add-currency", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleDevAddCurrency(writer, request, store)
	})
	mux.HandleFunc("/dev/items/set-level", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		handleDevSetLevel(writer, request, store)
	})
}

func resolveDevClassicIconDir() string {
	projectDir := strings.TrimSpace(os.Getenv("AI_PROJECT_DIR"))
	if projectDir == "" {
		projectDir = `D:\cocosProject\ai-project`
	}
	return filepath.Join(projectDir, "assets", "bundles", "shared", "runtime", "classic-icons", "icons")
}

func devItemsPageHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodGet {
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = devItemsPageTemplate.Execute(writer, nil)
}

func handleDevAddItem(writer http.ResponseWriter, request *http.Request, store *session.Store) {
	var payload devAddItemRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddItemResponse{
			Success:      false,
			ErrorCode:    "bad_json",
			ErrorMessage: "请求 JSON 不正确。",
		})
		return
	}
	payload.PlayerID = strings.TrimSpace(payload.PlayerID)
	payload.RoleID = strings.TrimSpace(payload.RoleID)
	payload.ItemID = strings.TrimSpace(payload.ItemID)
	if payload.PlayerID == "" || payload.RoleID == "" || payload.ItemID == "" {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddItemResponse{
			Success:      false,
			ErrorCode:    "missing_field",
			ErrorMessage: "需要 playerId、roleId 和 itemId。",
		})
		return
	}
	if payload.Count <= 0 {
		payload.Count = 1
	}

	item, ok := devCapturedRoleItemTemplateByID(payload.ItemID)
	if !ok {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddItemResponse{
			Success:      false,
			ErrorCode:    "item_not_found",
			ErrorMessage: "找不到这个道具 ID / 图标 ID / 名字。",
		})
		return
	}
	item.Type = "背包"
	item.Index = -1
	item.Count = payload.Count
	granted, ok := store.GrantRoleItem(payload.PlayerID, payload.RoleID, item)
	if !ok {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddItemResponse{
			Success:      false,
			ErrorCode:    "grant_failed",
			ErrorMessage: "添加失败，角色不存在或背包已满。",
		})
		return
	}

	role, ok := buildDevItemRole(store, payload.PlayerID, payload.RoleID)
	if !ok {
		writeDevJSON(writer, devAddItemResponse{Success: true, Item: granted})
		return
	}
	writeDevJSON(writer, devAddItemResponse{Success: true, Role: &role, Item: granted})
}

func handleDevAddCurrency(writer http.ResponseWriter, request *http.Request, store *session.Store) {
	var payload devAddCurrencyRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddCurrencyResponse{
			Success:      false,
			ErrorCode:    "bad_json",
			ErrorMessage: "请求 JSON 不正确。",
		})
		return
	}
	payload.PlayerID = strings.TrimSpace(payload.PlayerID)
	payload.RoleID = strings.TrimSpace(payload.RoleID)
	if payload.PlayerID == "" || payload.RoleID == "" {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddCurrencyResponse{
			Success:      false,
			ErrorCode:    "missing_field",
			ErrorMessage: "需要 playerId 和 roleId。",
		})
		return
	}
	if payload.Amount <= 0 {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddCurrencyResponse{
			Success:      false,
			ErrorCode:    "bad_amount",
			ErrorMessage: "元宝数量必须大于 0。",
		})
		return
	}

	currencies, ok := store.AddRoleCurrency(payload.PlayerID, payload.RoleID, "银元宝", payload.Amount)
	if !ok {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddCurrencyResponse{
			Success:      false,
			ErrorCode:    "add_currency_failed",
			ErrorMessage: "添加失败，角色不存在。",
		})
		return
	}
	if !syncDevSilverItemToCurrency(store, payload.PlayerID, payload.RoleID, currencies["银元宝"]) {
		writeDevJSONStatus(writer, http.StatusBadRequest, devAddCurrencyResponse{
			Success:      false,
			ErrorCode:    "sync_silver_item_failed",
			ErrorMessage: "元宝余额已增加，但背包元宝物品同步失败。",
			Currencies:   currencies,
		})
		return
	}
	role, ok := buildDevItemRole(store, payload.PlayerID, payload.RoleID)
	if !ok {
		writeDevJSON(writer, devAddCurrencyResponse{Success: true, Currencies: currencies})
		return
	}
	writeDevJSON(writer, devAddCurrencyResponse{Success: true, Role: &role, Currencies: currencies})
}

func handleDevSetLevel(writer http.ResponseWriter, request *http.Request, store *session.Store) {
	var payload devSetLevelRequest
	if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
		writeDevJSONStatus(writer, http.StatusBadRequest, devSetLevelResponse{
			Success:      false,
			ErrorCode:    "bad_json",
			ErrorMessage: "请求 JSON 不正确。",
		})
		return
	}
	payload.PlayerID = strings.TrimSpace(payload.PlayerID)
	payload.RoleID = strings.TrimSpace(payload.RoleID)
	if payload.PlayerID == "" || payload.RoleID == "" {
		writeDevJSONStatus(writer, http.StatusBadRequest, devSetLevelResponse{
			Success:      false,
			ErrorCode:    "missing_field",
			ErrorMessage: "需要 playerId 和 roleId。",
		})
		return
	}
	if payload.Level <= 0 {
		writeDevJSONStatus(writer, http.StatusBadRequest, devSetLevelResponse{
			Success:      false,
			ErrorCode:    "bad_level",
			ErrorMessage: "等级必须大于 0。",
		})
		return
	}

	result := store.SetRoleLevel(payload.PlayerID, payload.RoleID, payload.Level)
	if !result.Found || !result.Granted {
		writeDevJSONStatus(writer, http.StatusBadRequest, devSetLevelResponse{
			Success:      false,
			ErrorCode:    "set_level_failed",
			ErrorMessage: "设置失败，角色不存在。",
		})
		return
	}
	role, ok := buildDevItemRole(store, payload.PlayerID, payload.RoleID)
	if !ok {
		writeDevJSON(writer, devSetLevelResponse{Success: true})
		return
	}
	writeDevJSON(writer, devSetLevelResponse{Success: true, Role: &role})
}

func syncDevSilverItemToCurrency(store *session.Store, playerID string, roleID string, silverBalance int) bool {
	if silverBalance <= 0 {
		return true
	}
	items, _, ok := store.GetRoleItems(playerID, roleID, "背包")
	if !ok {
		return false
	}
	currentCount := 0
	for _, item := range items {
		if item.Name == "银元宝" {
			currentCount += item.Count
		}
	}
	missingCount := silverBalance - currentCount
	if missingCount <= 0 {
		return true
	}
	item, ok := session.CapturedRoleItemTemplate("银元宝")
	if !ok {
		return false
	}
	item.Type = "背包"
	item.Index = -1
	item.Count = missingCount
	_, ok = store.GrantRoleItem(playerID, roleID, item)
	return ok
}

func buildDevItemsState(store *session.Store) devItemsStateResponse {
	response := devItemsStateResponse{
		Roles:     []devItemRole{},
		Templates: devItemTemplates(),
	}
	for _, role := range store.DevRoleSummaries() {
		devRole, ok := buildDevItemRole(store, role.PlayerID, role.RoleID)
		if ok {
			response.Roles = append(response.Roles, devRole)
		}
	}
	return response
}

func buildDevItemRole(store *session.Store, playerID string, roleID string) (devItemRole, bool) {
	role, _, ok := store.GetRoleRuntimeData(playerID, roleID)
	if !ok {
		return devItemRole{}, false
	}
	items, capacity, ok := store.GetRoleItems(playerID, roleID, "背包")
	if !ok {
		return devItemRole{}, false
	}
	currencies, ok := store.GetRoleCurrencies(playerID, roleID)
	if !ok {
		currencies = session.RoleCurrencies{}
	}
	return devItemRole{
		PlayerID:    playerID,
		RoleID:      roleID,
		DisplayName: role.DisplayName,
		Level:       role.Level,
		Exp:         role.Exp,
		Vocation:    role.Voc,
		Currencies:  currencies,
		Items:       items,
		Capacity:    capacity,
	}, true
}

func devItemTemplates() []devItemTemplate {
	source := devCapturedRoleItemTemplates()
	result := make([]devItemTemplate, 0, len(source))
	for index, item := range source {
		result = append(result, devItemTemplate{
			ID:          strconv.Itoa(index + 1),
			Name:        item.Name,
			Display:     item.Display,
			ItemType:    item.ItemType,
			Description: item.Description,
		})
	}
	return result
}

func devCapturedRoleItemTemplateByID(itemID string) (session.RoleItem, bool) {
	itemID = strings.TrimSpace(strings.TrimSuffix(itemID, ".png"))
	if itemID == "" {
		return session.RoleItem{}, false
	}
	templates := devCapturedRoleItemTemplates()
	for index, item := range templates {
		if strconv.Itoa(index+1) == itemID {
			return item, true
		}
		if strings.TrimSuffix(item.Display, ".png") == itemID {
			return item, true
		}
		if item.Name == itemID {
			return item, true
		}
	}
	return session.RoleItem{}, false
}

func devCapturedRoleItemTemplates() []session.RoleItem {
	items := session.CapturedRoleItemTemplates()
	for _, handle := range []string{
		"7000542609490978",
		"4090542614314425",
		"4960542616750900",
		"1780542610743555",
		"4000542609162635",
		"1820542611400955",
		"1830542611405809",
		"2500542613172144",
		"2520542613299551",
	} {
		route, ok := sourceGuangqingItemShopRoutes[handle]
		if !ok {
			continue
		}
		for _, line := range strings.Split(strings.TrimSpace(route.rows), "\n") {
			row, ok := parseSourceItemShopRow(line)
			if ok {
				items = appendDevItemTemplate(items, sourceItemShopRowToRoleItem(row))
			}
		}
	}
	for _, answerHandle := range []string{"7", "8", "9"} {
		shop, ok := sourceSkillTeacherShops[answerHandle]
		if !ok {
			continue
		}
		for _, entry := range shop.Skills {
			items = appendDevItemTemplate(items, sourceSkillEntryToRoleItem(entry))
		}
	}
	return items
}

func appendDevItemTemplate(items []session.RoleItem, item session.RoleItem) []session.RoleItem {
	item.Name = strings.TrimSpace(item.Name)
	item.Display = strings.TrimSpace(item.Display)
	if item.Name == "" {
		return items
	}
	for _, existing := range items {
		if existing.Name == item.Name || (item.Display != "" && existing.Display == item.Display && existing.Name == item.Name) {
			return items
		}
	}
	return append(items, item)
}

func writeDevJSON(writer http.ResponseWriter, value any) {
	writeDevJSONStatus(writer, http.StatusOK, value)
}

func writeDevJSONStatus(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

var devItemsPageTemplate = template.Must(template.New("dev-items").Parse(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>道具发放</title>
  <style>
    :root { color-scheme: light; font-family: "Microsoft YaHei", Arial, sans-serif; background: #eef3f4; color: #1f2a2c; }
    body { margin: 0; padding: 24px; }
    h1 { margin: 0 0 18px; font-size: 24px; }
    h2 { margin: 0 0 12px; font-size: 17px; }
    .layout { display: grid; grid-template-columns: minmax(340px, 430px) minmax(520px, 1fr); gap: 18px; align-items: start; }
    section { background: #fff; border: 1px solid #cfd9dc; border-radius: 8px; padding: 16px; box-shadow: 0 2px 10px rgba(31, 42, 44, .06); }
    label { display: block; margin-bottom: 10px; font-size: 13px; color: #58676b; }
    input, select { width: 100%; box-sizing: border-box; margin-top: 5px; padding: 9px 10px; border: 1px solid #b9c7cb; border-radius: 6px; font-size: 14px; background: #fff; }
    button { width: 100%; border: 0; border-radius: 6px; padding: 10px 12px; background: #246b73; color: #fff; font-size: 15px; cursor: pointer; }
    button:hover { background: #1d5960; }
    button:disabled { background: #8da2a6; cursor: wait; }
    table { width: 100%; border-collapse: collapse; font-size: 13px; }
    th, td { border-bottom: 1px solid #e2e9eb; padding: 8px; text-align: left; vertical-align: top; }
    th { color: #54666b; background: #f7fafb; position: sticky; top: 0; }
    .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(210px, 1fr)); gap: 10px; max-height: 520px; overflow: auto; padding-right: 4px; }
    .card { border: 1px solid #d8e1e4; border-radius: 8px; padding: 10px; background: #fbfdfd; cursor: pointer; }
    .card:hover { border-color: #246b73; }
    .card-head { display: grid; grid-template-columns: 38px 1fr; gap: 8px; align-items: center; }
    .card strong { display: block; margin-bottom: 4px; font-size: 14px; }
    .icon { width: 32px; height: 32px; object-fit: contain; image-rendering: auto; vertical-align: middle; margin-right: 6px; }
    .icon-slot { width: 38px; height: 38px; display: inline-flex; align-items: center; justify-content: center; border: 1px solid #d8e1e4; border-radius: 6px; background: #f7fafb; }
    .muted { color: #6b7b80; font-size: 12px; }
    .status { margin-top: 10px; min-height: 20px; font-size: 13px; }
    .ok { color: #176b31; }
    .error { color: #b3261e; }
    .items { max-height: 420px; overflow: auto; border: 1px solid #e1e8ea; border-radius: 8px; }
    .toolbar { display: grid; grid-template-columns: 1fr 110px; gap: 8px; margin-bottom: 10px; }
    @media (max-width: 960px) { body { padding: 14px; } .layout { grid-template-columns: 1fr; } }
  </style>
</head>
<body>
  <h1>道具发放</h1>
  <div class="layout">
    <section>
      <h2>角色资源</h2>
      <label>角色
        <select id="roleSelect"></select>
      </label>
      <div id="currencyInfo" class="muted"></div>
      <label>设置等级
        <input id="levelInput" type="number" min="1" step="1" value="20">
      </label>
      <button id="setLevelButton">设置等级</button>
      <hr>
      <label>增加银元宝
        <input id="silverAmount" type="number" min="1" step="1" value="500">
      </label>
      <button id="addSilverButton">增加银元宝</button>
      <hr>
      <h2>添加道具</h2>
      <label>道具 ID / 图标 ID / 名字
        <input id="itemId" list="templateList" placeholder="例如 9、70、70.png、肉">
        <datalist id="templateList"></datalist>
      </label>
      <label>数量
        <input id="count" type="number" min="1" step="1" value="1">
      </label>
      <button id="addButton">添加到背包</button>
      <div id="status" class="status"></div>
    </section>
    <section>
      <div class="toolbar">
        <h2>当前背包</h2>
        <button id="refreshButton">刷新</button>
      </div>
      <div id="roleInfo" class="muted"></div>
      <div class="items">
        <table>
          <thead><tr><th>格子</th><th>名字</th><th>图标</th><th>数量</th><th>类型</th></tr></thead>
          <tbody id="itemsBody"></tbody>
        </table>
      </div>
    </section>
    <section style="grid-column: 1 / -1;">
      <h2>可添加道具</h2>
      <div id="templates" class="grid"></div>
    </section>
  </div>
  <script>
    const roleSelect = document.getElementById('roleSelect');
    const itemIdInput = document.getElementById('itemId');
    const countInput = document.getElementById('count');
    const levelInput = document.getElementById('levelInput');
    const silverAmountInput = document.getElementById('silverAmount');
    const statusEl = document.getElementById('status');
    const itemsBody = document.getElementById('itemsBody');
    const roleInfo = document.getElementById('roleInfo');
    const currencyInfo = document.getElementById('currencyInfo');
    const templatesEl = document.getElementById('templates');
    const templateList = document.getElementById('templateList');
    const addButton = document.getElementById('addButton');
    const addSilverButton = document.getElementById('addSilverButton');
    const setLevelButton = document.getElementById('setLevelButton');
    const refreshButton = document.getElementById('refreshButton');
    let state = { roles: [], templates: [] };

    function setStatus(text, isError) {
      statusEl.textContent = text || '';
      statusEl.className = 'status ' + (isError ? 'error' : 'ok');
    }

    function selectedRole() {
      return state.roles.find(role => role.roleId === roleSelect.value) || state.roles[0];
    }

    function iconUrl(display) {
      const name = String(display || '').trim().replace(/^\/+/, '');
      return name ? '/dev/classic-icons/' + encodeURIComponent(name) : '';
    }

    function createIcon(display) {
      const url = iconUrl(display);
      if (!url) return null;
      const image = document.createElement('img');
      image.className = 'icon';
      image.src = url;
      image.alt = display;
      image.loading = 'lazy';
      return image;
    }

    function renderRoles() {
      const previous = roleSelect.value;
      roleSelect.innerHTML = '';
      for (const role of state.roles) {
        const option = document.createElement('option');
        option.value = role.roleId;
        option.textContent = role.displayName + ' / ' + role.playerId + ' / ' + role.roleId;
        roleSelect.appendChild(option);
      }
      if (previous && state.roles.some(role => role.roleId === previous)) {
        roleSelect.value = previous;
      }
    }

    function renderItems() {
      const role = selectedRole();
      itemsBody.innerHTML = '';
      if (!role) {
        roleInfo.textContent = '没有角色。';
        return;
      }
      roleInfo.textContent = role.displayName + '  ' + (role.vocation || '新手') + ' Lv.' + role.level + '  ' + role.playerId + '  背包 ' + role.items.length + '/' + role.capacity;
      levelInput.value = role.level || 1;
      currencyInfo.textContent = '银元宝：' + (role.currencies?.银元宝 || 0) + '    铜钱：' + (role.currencies?.铜钱 || 0);
      for (const item of role.items) {
        const row = document.createElement('tr');
        row.innerHTML = '<td></td><td></td><td></td><td></td><td></td>';
        row.children[0].textContent = item.index;
        row.children[1].textContent = item.name;
        const icon = createIcon(item.display);
        if (icon) row.children[2].appendChild(icon);
        row.children[2].appendChild(document.createTextNode(item.display || ''));
        row.children[3].textContent = item.count;
        row.children[4].textContent = item.itemType;
        itemsBody.appendChild(row);
      }
    }

    function renderTemplates() {
      templatesEl.innerHTML = '';
      templateList.innerHTML = '';
      for (const item of state.templates) {
        const option = document.createElement('option');
        option.value = item.id;
        option.label = item.name + ' / ' + item.display;
        templateList.appendChild(option);

        const card = document.createElement('div');
        card.className = 'card';
        card.innerHTML = '<div class="card-head"><span class="icon-slot"></span><div><strong></strong><div class="muted"></div><div class="muted"></div></div></div>';
        const icon = createIcon(item.display);
        if (icon) card.querySelector('.icon-slot').appendChild(icon);
        card.querySelector('strong').textContent = '#' + item.id + ' ' + item.name;
        const meta = card.querySelectorAll('.muted');
        meta[0].textContent = '图标 ID: ' + item.display;
        meta[1].textContent = '类型: ' + item.itemType;
        card.addEventListener('click', () => {
          itemIdInput.value = item.id;
          setStatus('已选择 #' + item.id + ' ' + item.name, false);
        });
        templatesEl.appendChild(card);
      }
    }

    async function refresh() {
      setStatus('读取中...', false);
      const response = await fetch('/dev/items/state');
      state = await response.json();
      renderRoles();
      renderTemplates();
      renderItems();
      setStatus('已刷新。', false);
    }

    async function addItem() {
      const role = selectedRole();
      if (!role) {
        setStatus('没有可用角色。', true);
        return;
      }
      addButton.disabled = true;
      try {
        const response = await fetch('/dev/items/add', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            playerId: role.playerId,
            roleId: role.roleId,
            itemId: itemIdInput.value,
            count: Number(countInput.value || 1),
          }),
        });
        const result = await response.json();
        if (!result.success) {
          setStatus(result.errorMessage || '添加失败。', true);
          return;
        }
        const index = state.roles.findIndex(current => current.roleId === result.role.roleId);
        if (index >= 0) state.roles[index] = result.role;
        renderItems();
        setStatus('已添加：' + result.item.name + ' x' + result.item.count + '。', false);
      } finally {
        addButton.disabled = false;
      }
    }

    async function addSilver() {
      const role = selectedRole();
      if (!role) {
        setStatus('没有可用角色。', true);
        return;
      }
      addSilverButton.disabled = true;
      try {
        const response = await fetch('/dev/items/add-currency', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            playerId: role.playerId,
            roleId: role.roleId,
            amount: Number(silverAmountInput.value || 1),
          }),
        });
        const result = await response.json();
        if (!result.success) {
          setStatus(result.errorMessage || '增加银元宝失败。', true);
          return;
        }
        const index = state.roles.findIndex(current => current.roleId === result.role.roleId);
        if (index >= 0) state.roles[index] = result.role;
        renderItems();
        setStatus('已增加银元宝，当前：' + result.currencies.银元宝 + '。', false);
      } finally {
        addSilverButton.disabled = false;
      }
    }

    async function setLevel() {
      const role = selectedRole();
      if (!role) {
        setStatus('没有可用角色。', true);
        return;
      }
      setLevelButton.disabled = true;
      try {
        const response = await fetch('/dev/items/set-level', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            playerId: role.playerId,
            roleId: role.roleId,
            level: Number(levelInput.value || 1),
          }),
        });
        const result = await response.json();
        if (!result.success) {
          setStatus(result.errorMessage || '设置等级失败。', true);
          return;
        }
        const index = state.roles.findIndex(current => current.roleId === result.role.roleId);
        if (index >= 0) state.roles[index] = result.role;
        renderItems();
        setStatus('已设置等级：Lv.' + result.role.level + '。', false);
      } finally {
        setLevelButton.disabled = false;
      }
    }

    roleSelect.addEventListener('change', renderItems);
    refreshButton.addEventListener('click', refresh);
    addButton.addEventListener('click', addItem);
    addSilverButton.addEventListener('click', addSilver);
    setLevelButton.addEventListener('click', setLevel);
    refresh().catch(error => setStatus(String(error), true));
  </script>
</body>
</html>`))
