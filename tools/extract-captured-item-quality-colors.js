#!/usr/bin/env node

/*
 * Updates the Classic item table from raw captured f_i_<name>^RRGGBB metadata.
 * Items without an explicit source color retain the Flash client default: white.
 */
const fs = require("fs");
const path = require("path");

const serverRoot = path.resolve(__dirname, "..");
const projectRoot = "D:/cocosProject/ai-project";
const tablePath = path.join(serverRoot, "internal/classicdata/source/item-table.csv");
const reportPath = path.join(projectRoot, "tmp/classic-item-quality-colors-report.json");
const captureRoots = [
  "D:/yzhgame/WOCClient/tmp/woc-proxy-captures",
  "D:/yzhgame/WOCClient/instances",
  path.join(projectRoot, "tmp"),
];
const qualityColorPattern = /f_i_([^\^&\0\r\n]{1,120})\^([0-9a-fA-F]{6})/g;
const qualityColorPrefixPattern = /^(f_i_[^^&]+)(?:\^([0-9a-fA-F]{6}))?/;
const supportedCaptureName = /(?:server-to-client.*\.bin|frames-.*\.ndjson)$/i;
const requestedWrite = process.argv.includes("--write");

function parseCsv(text) {
  const rows = [];
  let row = [];
  let field = "";
  let quoted = false;
  for (let index = 0; index < text.length; index += 1) {
    const character = text[index];
    if (quoted) {
      if (character === '"') {
        if (text[index + 1] === '"') {
          field += '"';
          index += 1;
        } else {
          quoted = false;
        }
      } else {
        field += character;
      }
      continue;
    }
    if (character === '"') {
      quoted = true;
    } else if (character === ",") {
      row.push(field);
      field = "";
    } else if (character === "\n") {
      if (field.endsWith("\r")) {
        field = field.slice(0, -1);
      }
      row.push(field);
      rows.push(row);
      row = [];
      field = "";
    } else {
      field += character;
    }
  }
  if (field.length > 0 || row.length > 0) {
    row.push(field);
    rows.push(row);
  }
  return rows;
}

function formatCsvField(value) {
  return '"' + String(value || "").replace(/"/g, '""') + '"';
}

function walkCaptureFiles(root, files) {
  if (!fs.existsSync(root)) {
    return;
  }
  for (const entry of fs.readdirSync(root, { withFileTypes: true })) {
    const entryPath = path.join(root, entry.name);
    if (entry.isDirectory()) {
      walkCaptureFiles(entryPath, files);
    } else if (entry.isFile() && supportedCaptureName.test(entry.name)) {
      files.push(entryPath);
    }
  }
}

function decodedCandidates(buffer) {
  const candidates = [buffer.toString("utf8")];
  try {
    candidates.push(new TextDecoder("gb18030").decode(buffer));
  } catch (error) {
    // The raw protocol is normally UTF-8; GB18030 is only for older capture files.
  }
  return candidates;
}

function findCapturedColors(files, knownNames) {
  const captured = new Map();
  for (const filePath of files) {
    const contents = fs.readFileSync(filePath);
    for (const text of decodedCandidates(contents)) {
      qualityColorPattern.lastIndex = 0;
      let match = qualityColorPattern.exec(text);
      while (match) {
        const name = match[1].trim();
        if (knownNames.has(name)) {
          const color = match[2].toLowerCase();
          let entry = captured.get(name);
          if (!entry) {
            entry = { colors: new Set(), evidencePaths: new Map() };
            captured.set(name, entry);
          }
          entry.colors.add(color);
          if (!entry.evidencePaths.has(color)) {
            entry.evidencePaths.set(color, filePath.replace(/\\/g, "/"));
          }
        }
        match = qualityColorPattern.exec(text);
      }
    }
  }
  return captured;
}

function main() {
  const csvRows = parseCsv(fs.readFileSync(tablePath, "utf8").replace(/^\uFEFF/, ""));
  if (csvRows.length < 2) {
    throw new Error("item table is empty");
  }
  const headers = csvRows[0];
  const nameIndex = headers.indexOf("name");
  if (nameIndex < 0) {
    throw new Error("item table does not contain a name column");
  }
  let colorIndex = headers.indexOf("quality_color");
  if (colorIndex < 0) {
    colorIndex = headers.length;
    headers.push("quality_color");
  }
  let evidenceIndex = headers.indexOf("quality_color_evidence");
  if (evidenceIndex < 0) {
    evidenceIndex = headers.length;
    headers.push("quality_color_evidence");
  }

  const knownNames = new Set(csvRows.slice(1).map((row) => String(row[nameIndex] || "").trim()).filter(Boolean));
  const files = [];
  for (const root of captureRoots) {
    walkCaptureFiles(root, files);
  }
  const captured = findCapturedColors(files, knownNames);
  const conflicts = [];
  for (const [name, entry] of captured) {
    if (entry.colors.size > 1) {
      conflicts.push({ name, colors: Array.from(entry.colors).sort() });
    }
  }
  if (conflicts.length > 0) {
    throw new Error("captured item quality color conflicts: " + JSON.stringify(conflicts));
  }

  const items = [];
  let capturedCount = 0;
  let sourceDefaultWhiteCount = 0;
  for (const row of csvRows.slice(1)) {
    while (row.length < headers.length) {
      row.push("");
    }
    const name = String(row[nameIndex] || "").trim();
    const entry = captured.get(name);
    if (entry) {
      const color = Array.from(entry.colors)[0];
      const evidencePath = entry.evidencePaths.get(color);
      row[colorIndex] = color;
      row[evidenceIndex] = "captured:" + evidencePath;
      row[headers.indexOf("description")] = applyQualityColorToDescription(row[headers.indexOf("description")], color);
      capturedCount += 1;
      items.push({ name, quality_color: color, quality_color_evidence: row[evidenceIndex] });
    } else {
      row[colorIndex] = "ffffff";
      row[evidenceIndex] = "source_default_white";
      row[headers.indexOf("description")] = applyQualityColorToDescription(row[headers.indexOf("description")], "ffffff");
      sourceDefaultWhiteCount += 1;
      items.push({ name, quality_color: "ffffff", quality_color_evidence: "source_default_white" });
    }
  }

  const focusNames = ["宝匣", "魔匣", "仙匣", "凿孔器", "精炼宝石", "贰级原石", "高级精炼宝石"];
  const report = {
    generated_at: new Date().toISOString(),
    table: tablePath.replace(/\\/g, "/"),
    capture_roots: captureRoots,
    files_scanned: files.length,
    item_count: items.length,
    explicit_captured_color_count: capturedCount,
    source_default_white_count: sourceDefaultWhiteCount,
    conflict_count: conflicts.length,
    focus_items: items.filter((item) => focusNames.includes(item.name)),
    items,
  };
  fs.mkdirSync(path.dirname(reportPath), { recursive: true });
  fs.writeFileSync(reportPath, JSON.stringify(report, null, 2) + "\n", "utf8");

  if (requestedWrite) {
    const output = [headers.join(",")]
      .concat(csvRows.slice(1).map((row) => row.map(formatCsvField).join(",")))
      .join("\r\n") + "\r\n";
    fs.writeFileSync(tablePath, output, "utf8");
  }
  console.log(JSON.stringify({
    files_scanned: files.length,
    item_count: items.length,
    explicit_captured_color_count: capturedCount,
    source_default_white_count: sourceDefaultWhiteCount,
    conflict_count: conflicts.length,
    report: reportPath.replace(/\\/g, "/"),
    wrote: requestedWrite,
    focus_items: report.focus_items,
  }, null, 2));
}

function applyQualityColorToDescription(description, color) {
  const source = String(description || "");
  const match = qualityColorPrefixPattern.exec(source);
  if (!match) {
    return source;
  }
  return match[1] + "^" + color + source.slice(match[0].length);
}

main();
