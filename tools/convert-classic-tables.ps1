param(
  [string]$ServerRoot = ''
)

if ($PSVersionTable.PSVersion.Major -lt 6) {
  $pwsh = Get-Command pwsh -ErrorAction SilentlyContinue
  if ($null -eq $pwsh) {
    throw 'convert-classic-tables.ps1 requires PowerShell 7 or newer for stable generated JSON output.'
  }
  $pwshArguments = @('-NoProfile', '-File', $PSCommandPath)
  if (![string]::IsNullOrWhiteSpace($ServerRoot)) {
    $pwshArguments += @('-ServerRoot', $ServerRoot)
  }
  & $pwsh.Source @pwshArguments
  exit $LASTEXITCODE
}

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ([string]::IsNullOrWhiteSpace($ServerRoot)) {
  $ServerRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
}

$sourceRoot = Join-Path $ServerRoot 'internal\classicdata\source'
$generatedRoot = Join-Path $ServerRoot 'internal\classicdata\generated'

if (!(Test-Path $sourceRoot)) {
  throw "Classic data source directory not found: $sourceRoot"
}
if (!(Test-Path $generatedRoot)) {
  New-Item -ItemType Directory -Path $generatedRoot | Out-Null
}

$tables = @(
  @{
    Name = 'drop'
    Source = 'drop-table.csv'
    RequiredColumns = @('map_id', 'source_monster_handle', 'exp_delta', 'item_observed_rates', 'status')
  },
  @{
    Name = 'item'
    Source = 'item-table.csv'
    RequiredColumns = @('status', 'confidence', 'icon', 'name', 'item_type', 'category', 'quality_color', 'quality_color_evidence')
  },
  @{
    Name = 'fashion-appearance'
    Source = 'fashion-appearance-table.csv'
    RequiredColumns = @('fashion_name', 'sex', 'source_params', 'evidence_status')
  },
  @{
    Name = 'equipment-appearance'
    Source = 'equipment-appearance-table.csv'
    RequiredColumns = @('equipment_name', 'source_key', 'source_value', 'evidence_status')
  },
  @{
    Name = 'skill'
    Source = 'skill-table.csv'
    RequiredColumns = @('skill_id', 'kind', 'label', 'source_type', 'action_name', 'target')
  },
  @{
    Name = 'skill-level'
    Source = 'skill-level-table.csv'
    RequiredColumns = @('skill_level_id', 'skill_id', 'level', 'source_action_label', 'damage_multiplier', 'mp_cost')
  },
  @{
    Name = 'monster-skill'
    Source = 'monster-skill-table.csv'
    RequiredColumns = @('monster_skill_id', 'command_id', 'monster_name', 'display_url', 'action_name', 'source_type', 'source_action_label', 'target', 'runtime_chance_percent', 'evidence_status')
  },
  @{
    Name = 'profession'
    Source = 'profession-table.csv'
    RequiredColumns = @('profession_id', 'name', 'answer_handle', 'skill_shop_id', 'skill_cap')
  },
  @{
    Name = 'buff'
    Source = 'buff-table.csv'
    RequiredColumns = @('buff_id', 'name', 'display', 'status', 'mechanism')
  },
  @{
    Name = 'monster'
    Source = 'monster-table.csv'
    RequiredColumns = @('monster_id', 'source_kind', 'map_id', 'handle', 'name', 'display_url', 'level', 'vocation', 'max_hp', 'max_mp')
  },
  @{
    Name = 'attribute'
    Source = 'attribute-table.csv'
    RequiredColumns = @('attribute_id', 'display_name', 'kind', 'scope', 'status')
  },
  @{
    Name = 'effect'
    Source = 'effect-table.csv'
    RequiredColumns = @('effect_id', 'buff_id', 'attribute_id', 'operation', 'status')
  },
  @{
    Name = 'effect-source'
    Source = 'effect-source-table.csv'
    RequiredColumns = @('source_link_id', 'source_type', 'source_id', 'trigger', 'target', 'buff_id', 'status')
  }
)

function Test-RequiredColumns {
  param(
    [string]$Path,
    [object[]]$Rows,
    [string[]]$RequiredColumns
  )

  if ($Rows.Count -le 0) {
    throw "Source table has no data rows: $Path"
  }

  $headers = @($Rows[0].PSObject.Properties.Name)
  foreach ($column in $RequiredColumns) {
    if ($headers -notcontains $column) {
      throw "Source table $Path missing required column: $column"
    }
  }
}

foreach ($table in $tables) {
  $sourcePath = Join-Path $sourceRoot $table.Source
  if (!(Test-Path $sourcePath)) {
    throw "Source CSV not found: $sourcePath"
  }

  $rows = @(Import-Csv -LiteralPath $sourcePath)
  Test-RequiredColumns -Path $sourcePath -Rows $rows -RequiredColumns $table.RequiredColumns

  $relativeSource = "internal/classicdata/source/$($table.Source)"
  $payload = [ordered]@{
    schemaVersion = 1
    name = $table.Name
    source = $relativeSource
    rowCount = $rows.Count
    rows = $rows
  }

  $destinationPath = Join-Path $generatedRoot "$($table.Name)-table.json"
  $json = $payload | ConvertTo-Json -Depth 32
  [System.IO.File]::WriteAllText($destinationPath, $json + "`n", [System.Text.UTF8Encoding]::new($false))
  Write-Host "Generated $destinationPath from $sourcePath ($($rows.Count) rows)"
}
