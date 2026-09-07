const attributeNames = [
  '物理攻击', '魔法攻击', '物理防御', '魔法防御', '气力上限', '精力上限',
  '恢复气力', '恢复精力', '命中', '回避', '爆击', '移动', '力量', '智力', '敏捷', '耐力', '幸运',
];

function valueUnit(value) {
  if (/^[+-]?\d+(?:\.\d+)?%$/.test(value)) return 'percent';
  if (/^[+-]?\d+(?:\.\d+)?$/.test(value)) return 'flat';
  return value.includes('$') ? 'expression' : 'text';
}

function equipmentAttributes(fields, splitValue) {
  return attributeNames.flatMap((name, index) => {
    const code = String(index + 1);
    if (!(code in fields) || fields[code] === '') return [];
    const parsed = splitValue(fields[code]);
    return [{ code, name, value: parsed.base, unit: valueUnit(parsed.base), capturedBonus: parsed.bonus }];
  });
}

function plainText(value) {
  return value.replace(/<br\s*\/?\s*>/gi, '\n').replace(/<[^>]*>/g, '').trim();
}

function refinementSteps(text) {
  const steps = [];
  // Short [+N] Web descriptions have a different contract. Keep their text
  // without assuming the Classic "every level from this threshold" operation.
  const pattern = /\[(精炼)?\+(\d+)\]([^\r\n]*)/g;
  for (const match of text.matchAll(pattern)) {
    const sourceText = match[0];
    const body = plainText(match[3]);
    const step = {
      level: Number(match[2]), syntax: match[1] ? 'classic' : 'short',
      kind: 'text_only', effects: [], sourceText,
    };
    if (match[1] && body.startsWith('每升一级 ')) {
      const effectText = body.slice('每升一级 '.length).trim();
      const effectPattern = /([^+\d]+)\+(\d+(?:\.\d+)?)(%?)(?:\s+|$)/g;
      let consumed = 0;
      for (const effect of effectText.matchAll(effectPattern)) {
        if (effect.index !== consumed) break;
        const name = effect[1].trim();
        const index = attributeNames.indexOf(name);
        step.effects.push({ code: index >= 0 ? String(index + 1) : '', name, amount: Number(effect[2]), unit: effect[3] ? 'percent' : 'flat' });
        consumed = effect.index + effect[0].length;
      }
      if (consumed === effectText.length && step.effects.length) step.kind = 'per_level';
      else step.effects = [];
    }
    steps.push(step);
  }
  return steps;
}

function effectText(fields) {
  return fields.filter(field => field.code === '19').map(field => field.value
    .replace(/\[(?:精炼)?\+\d+\][^\r\n]*/g, '')
    .replace(/精炼潜质\s*[:：]?/g, ''))
    .map(plainText).filter(Boolean).join('\n');
}

module.exports = { equipmentAttributes, refinementSteps, effectText, valueUnit };
