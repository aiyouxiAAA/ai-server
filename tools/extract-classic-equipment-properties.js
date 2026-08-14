#!/usr/bin/env node

/*
 * Extracts structured equipment properties from the first f_i_ segment of
 * item-table descriptions. Later f_i_ segments are inserted gems; fields
 * after the refinement block belong to a captured item instance, not the
 * reusable equipment template.
 */
const fs = require("fs");
const path = require("path");

const serverRoot = path.resolve(__dirname, "..");
const itemTablePath = path.join(serverRoot, "internal/classicdata/source/item-table.csv");
const writeRequested = process.argv.includes("--write");

const propertyHeaders = [
  "required_level",
  "required_vocation",
  "required_prestige",
  "required_crime",
  "socket_capacity",
  "phy_atk",
  "mgc_atk",
  "phy_def",
  "mgc_def",
  "max_hp",
  "max_mp",
  "hit",
  "dodge",
  "crit",
  "move",
  "strength",
  "intelligence",
  "agility",
  "constitution",
  "luck",
  "captured_instance_attributes",
  "captured_instance_bonus_attributes",
  "special_effects",
  "refinement_rules",
  "property_parse_status",
];

const sourceFieldToColumn = {
  1: "phy_atk",
  2: "mgc_atk",
  3: "phy_def",
  4: "mgc_def",
  5: "max_hp",
  6: "max_mp",
  9: "hit",
  10: "dodge",
  11: "crit",
  12: "move",
  13: "strength",
  14: "intelligence",
  15: "agility",
  16: "constitution",
  17: "luck",
  18: "socket_capacity",
  21: "required_level",
  22: "required_vocation",
  30: "required_prestige",
  31: "required_crime",
};

function parseCsv(text) {
  const rows = [];
  let row = [];
  let field = "";
  let quoted = false;
  for (let index = 0; index < text.length; index += 1) {
    const char = text[index];
    if (quoted) {
      if (char === '"') {
        if (text[index + 1] === '"') {
          field += '"';
          index += 1;
        } else {
          quoted = false;
        }
      } else {
        field += char;
      }
      continue;
    }
    if (char === '"') {
      quoted = true;
    } else if (char === ",") {
      row.push(field);
      field = "";
    } else if (char === "\n") {
      row.push(field.endsWith("\r") ? field.slice(0, -1) : field);
      rows.push(row);
      row = [];
      field = "";
    } else {
      field += char;
    }
  }
  if (field.length > 0 || row.length > 0) {
    row.push(field);
    rows.push(row);
  }
  return rows;
}

function readCsv(pathname) {
  const matrix = parseCsv(fs.readFileSync(pathname, "utf8").replace(/^\uFEFF/, ""));
  const headers = matrix.shift();
  return {
    headers,
    rows: matrix.filter((row) => row.length > 1 || row[0] !== "").map((row) => {
      const result = {};
      headers.forEach((header, index) => {
        result[header] = (row[index] || "").replace(/\r\n?/g, "\n");
      });
      return result;
    }),
  };
}

function csvValue(value) {
  return '"' + String(value || "").replace(/"/g, '""') + '"';
}

function writeCsv(pathname, headers, rows) {
  const output = [headers.map(csvValue).join(",")]
    .concat(rows.map((row) => headers.map((header) => csvValue(row[header])).join(",")))
    .join("\n") + "\n";
  fs.writeFileSync(pathname, output, "utf8");
}

function primaryItemSegment(description) {
  const start = description.indexOf("f_i_");
  if (start < 0) {
    return "";
  }
  const next = description.indexOf("f_i_", start + 4);
  return description.slice(start, next >= 0 ? next : description.length);
}

function descriptionFields(text) {
  const markers = [];
  const pattern = /&(\d+)@/g;
  let match = pattern.exec(text);
  while (match) {
    markers.push({ code: match[1], start: match.index, valueStart: pattern.lastIndex });
    match = pattern.exec(text);
  }
  return markers.map((marker, index) => ({
    code: marker.code,
    value: text.slice(marker.valueStart, index + 1 < markers.length ? markers[index + 1].start : text.length).trim(),
  }));
}

function firstFieldsByCode(fields) {
  const result = {};
  for (const field of fields) {
    if (!(field.code in result)) {
      result[field.code] = field.value;
    }
  }
  return result;
}

function splitBaseValueAndInstanceBonus(value) {
  const normalized = String(value || "").trim();
  const match = /^([^()]*)(?:\(([^()]*)\))?$/.exec(normalized);
  if (!match) {
    return { base: normalized, bonus: "" };
  }
  return { base: match[1].trim(), bonus: (match[2] || "").trim() };
}

function decodeFlashMarkup(text) {
  return text
    .replaceAll("&0;", ",")
    .replaceAll("&1;", "@")
    .replaceAll("&2;", "<")
    .replaceAll("&3;", ">");
}

function extractSpecialEffects(refinementRules) {
  const normalized = decodeFlashMarkup(refinementRules);
  const effectMarker = normalized.match(/特殊(?:效果|功能):/);
  if (!effectMarker || effectMarker.index === undefined) {
    return "";
  }
  let effect = normalized.slice(effectMarker.index + effectMarker[0].length);
  const close = effect.indexOf("</font>");
  if (close >= 0) {
    effect = effect.slice(0, close);
  }
  const nextRefinement = effect.search(/\[精炼\+/);
  if (nextRefinement >= 0) {
    effect = effect.slice(0, nextRefinement);
  }
  return effect.replace(/<[^>]+>/g, "").trim();
}

function serializeAttributes(attributes) {
  const entries = Object.entries(attributes);
  return entries.length > 0 ? JSON.stringify(Object.fromEntries(entries)) : "";
}

function enrichEquipmentRow(row) {
  for (const header of propertyHeaders) {
    row[header] = "";
  }
  if (row.item_type !== "equip") {
    row.property_parse_status = "not_equipment";
    return { status: row.property_parse_status, instance: false, variables: false };
  }

  const primary = primaryItemSegment(row.description || "");
  if (!primary) {
    row.property_parse_status = "missing_f_i";
    return { status: row.property_parse_status, instance: false, variables: false };
  }

  const refinementOffset = primary.indexOf("&19@");
  const templateFields = firstFieldsByCode(descriptionFields(
    refinementOffset >= 0 ? primary.slice(0, refinementOffset) : primary,
  ));
  const allFields = descriptionFields(primary);
  const instanceFields = refinementOffset < 0
    ? []
    : allFields.slice(allFields.findIndex((field) => field.code === "19") + 1);
  const capturedInstanceAttributes = {};
  const capturedInstanceBonuses = {};
  let hasVariables = false;

  for (const [sourceCode, column] of Object.entries(sourceFieldToColumn)) {
    const raw = templateFields[sourceCode];
    if (raw === undefined) {
      continue;
    }
    const parsed = splitBaseValueAndInstanceBonus(raw);
    row[column] = parsed.base;
    if (parsed.bonus) {
      capturedInstanceBonuses[column] = parsed.bonus;
    }
    if (raw.includes("$")) {
      hasVariables = true;
    }
  }

  for (const field of instanceFields) {
    const column = sourceFieldToColumn[field.code];
    if (!column || field.value === "") {
      continue;
    }
    capturedInstanceAttributes[column] = field.value;
    const parsed = splitBaseValueAndInstanceBonus(field.value);
    if (parsed.bonus) {
      capturedInstanceBonuses[column] = parsed.bonus;
    }
  }

  const refinementField = allFields.find((field) => field.code === "19");
  row.refinement_rules = refinementField ? refinementField.value : "";
  row.special_effects = extractSpecialEffects(row.refinement_rules);
  row.captured_instance_attributes = serializeAttributes(capturedInstanceAttributes);
  row.captured_instance_bonus_attributes = serializeAttributes(capturedInstanceBonuses);

  const hasInstance = Object.keys(capturedInstanceAttributes).length > 0
    || Object.keys(capturedInstanceBonuses).length > 0;
  row.property_parse_status = hasVariables ? "template_with_variables" : "template_static";
  if (hasInstance) {
    row.property_parse_status += "_with_captured_instance";
  }
  return { status: row.property_parse_status, instance: hasInstance, variables: hasVariables };
}

function main() {
  const table = readCsv(itemTablePath);
  const headers = table.headers.slice();
  for (const header of propertyHeaders) {
    if (!headers.includes(header)) {
      headers.push(header);
    }
  }

  let equipmentRows = 0;
  let instanceRows = 0;
  let variableRows = 0;
  const statusCounts = {};
  for (const row of table.rows) {
    const result = enrichEquipmentRow(row);
    statusCounts[result.status] = (statusCounts[result.status] || 0) + 1;
    if (row.item_type === "equip") {
      equipmentRows += 1;
    }
    if (result.instance) {
      instanceRows += 1;
    }
    if (result.variables) {
      variableRows += 1;
    }
  }

  const report = {
    item_table: itemTablePath.replace(/\\/g, "/"),
    rows: table.rows.length,
    equipment_rows: equipmentRows,
    equipment_with_captured_instance_values: instanceRows,
    equipment_with_template_variables: variableRows,
    property_headers: propertyHeaders,
    property_parse_status_counts: statusCounts,
    write_requested: writeRequested,
  };
  if (writeRequested) {
    writeCsv(itemTablePath, headers, table.rows);
  }
  console.log(JSON.stringify(report, null, 2));
}

main();
