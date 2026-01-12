/**
 * Generate Zod schemas from WebSocket YAML schema files.
 * 
 * Usage: bun run scripts/generate-ws-schemas.ts
 */

import { readFileSync, writeFileSync, mkdirSync } from "fs";
import { parse as parseYaml } from "yaml";
import { join, dirname } from "path";

const API_WS_DIR = join(dirname(dirname(__dirname)), "api", "ws");
const OUTPUT_DIR = join(dirname(__dirname), "src", "model", "ws");

// Schema files to process
const SCHEMA_FILES = [
  { input: "messages.schema.yaml", output: "messages.ts" },
  { input: "subscription.schema.yaml", output: "subscription.ts" },
  { input: "project.schema.yaml", output: "project.ts" },
];

function toSchemaName(name: string): string {
  // Convert PascalCase to camelCase and add Schema suffix
  return name.charAt(0).toLowerCase() + name.slice(1) + "Schema";
}

function jsonSchemaToZod(schema: any, definitions: Record<string, any>, indent = 0): string {
  const pad = "  ".repeat(indent);
  
  if (!schema) return "z.unknown()";
  
  // Handle $ref
  if (schema.$ref) {
    const refName = schema.$ref.replace("#/definitions/", "");
    return toSchemaName(refName);
  }
  
  // Handle enum
  if (schema.enum) {
    const values = schema.enum.map((v: string) => `"${v}"`).join(", ");
    return `z.enum([${values}])`;
  }
  
  // Handle type
  switch (schema.type) {
    case "string":
      let str = "z.string()";
      if (schema.format === "uuid") str = "z.string().uuid()";
      else if (schema.format === "date-time") str = "z.string().datetime({ offset: true })";
      return str;
    
    case "integer":
      return "z.number().int()";
    
    case "number":
      return "z.number()";
    
    case "boolean":
      return "z.boolean()";
    
    case "array":
      const itemsZod = jsonSchemaToZod(schema.items, definitions, indent);
      return `z.array(${itemsZod})`;
    
    case "object":
      if (schema.additionalProperties) {
        // Record type
        if (schema.additionalProperties === true) {
          return "z.record(z.unknown())";
        }
        const valueZod = jsonSchemaToZod(schema.additionalProperties, definitions, indent);
        return `z.record(${valueZod})`;
      }
      
      if (!schema.properties) {
        return "z.object({})";
      }
      
      const required = new Set(schema.required || []);
      const props: string[] = [];
      
      for (const [key, prop] of Object.entries(schema.properties as Record<string, any>)) {
        let propZod = jsonSchemaToZod(prop, definitions, indent + 1);
        if (!required.has(key)) {
          propZod += ".optional()";
        }
        props.push(`${pad}  ${key}: ${propZod}`);
      }
      
      if (props.length === 0) {
        return "z.object({})";
      }
      
      return `z.object({\n${props.join(",\n")}\n${pad}})`;
    
    default:
      // No type specified - might have other constraints
      if (schema.properties) {
        return jsonSchemaToZod({ ...schema, type: "object" }, definitions, indent);
      }
      return "z.unknown()";
  }
}

function generateSchemaFile(inputFile: string, outputFile: string): void {
  console.log(`Processing ${inputFile}...`);
  
  const inputPath = join(API_WS_DIR, inputFile);
  const outputPath = join(OUTPUT_DIR, outputFile);
  
  // Read and parse YAML
  const yamlContent = readFileSync(inputPath, "utf-8");
  const schema = parseYaml(yamlContent);
  
  if (!schema.definitions) {
    console.log(`  No definitions found in ${inputFile}, skipping.`);
    return;
  }
  
  const lines: string[] = [
    "/**",
    ` * Generated from ${inputFile}`,
    " * Do not edit manually.",
    " */",
    "",
    'import { z } from "zod";',
    "",
  ];
  
  // Determine dependency order using topological sort
  const definitions = schema.definitions as Record<string, any>;
  const definitionOrder: string[] = [];
  const visited = new Set<string>();
  const visiting = new Set<string>();
  
  function getDependencies(def: any): string[] {
    const deps: string[] = [];
    const json = JSON.stringify(def);
    const refMatches = json.matchAll(/#\/definitions\/(\w+)/g);
    for (const match of refMatches) {
      if (definitions[match[1]]) {
        deps.push(match[1]);
      }
    }
    return [...new Set(deps)];
  }
  
  function visit(name: string) {
    if (visited.has(name)) return;
    if (visiting.has(name)) {
      // Circular dependency detected, will use z.lazy
      return;
    }
    
    visiting.add(name);
    const def = definitions[name];
    if (def) {
      for (const dep of getDependencies(def)) {
        visit(dep);
      }
    }
    visiting.delete(name);
    visited.add(name);
    definitionOrder.push(name);
  }
  
  for (const name of Object.keys(definitions)) {
    visit(name);
  }
  
  // Generate schemas in dependency order
  for (const name of definitionOrder) {
    const def = definitions[name];
    const schemaName = toSchemaName(name);
    
    try {
      const zodCode = jsonSchemaToZod(def, definitions);
      const description = def.description ? `.describe("${def.description.replace(/"/g, '\\"')}")` : "";
      
      lines.push(`// ${name}`);
      lines.push(`export const ${schemaName} = ${zodCode}${description};`);
      lines.push(`export type ${name} = z.infer<typeof ${schemaName}>;`);
      lines.push("");
    } catch (error) {
      console.error(`  Error generating schema for ${name}:`, error);
    }
  }
  
  // Generate message type constants if messageTypes exists
  if (schema.messageTypes) {
    lines.push("// Message type constants");
    lines.push("export const MessageTypes = {");
    
    for (const msgType of Object.keys(schema.messageTypes as Record<string, any>)) {
      // Convert message.type.name to MESSAGE_TYPE_NAME
      const constName = msgType.toUpperCase().replace(/\./g, "_").replace(/-/g, "_");
      lines.push(`  ${constName}: "${msgType}",`);
    }
    
    lines.push("} as const;");
    lines.push("");
    lines.push("export type MessageType = (typeof MessageTypes)[keyof typeof MessageTypes];");
    lines.push("");
  }
  
  // Write output
  mkdirSync(dirname(outputPath), { recursive: true });
  writeFileSync(outputPath, lines.join("\n"));
  console.log(`  Generated ${outputFile} with ${definitionOrder.length} schemas`);
}

function generateIndexFile(): void {
  const indexPath = join(OUTPUT_DIR, "index.ts");
  const lines = [
    "/**",
    " * WebSocket schema exports",
    " * Generated - do not edit manually.",
    " */",
    "",
  ];
  
  for (const { output } of SCHEMA_FILES) {
    const moduleName = output.replace(".ts", "");
    lines.push(`export * from "./${moduleName}";`);
  }
  
  writeFileSync(indexPath, lines.join("\n"));
  console.log("Generated index.ts");
}

// Main
console.log("Generating WebSocket Zod schemas...\n");

mkdirSync(OUTPUT_DIR, { recursive: true });

for (const { input, output } of SCHEMA_FILES) {
  try {
    generateSchemaFile(input, output);
  } catch (error) {
    console.error(`Failed to process ${input}:`, error);
  }
}

generateIndexFile();

console.log("\nDone!");
