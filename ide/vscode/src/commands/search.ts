import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { formatBytes, requireConnection, withProgress } from './helpers';

export function registerSearchCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  // Find in codebase command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.find', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const query = await vscode.window.showInputBox({
        prompt: 'Enter search query (regex supported)',
        placeHolder: 'Search pattern...',
      });

      if (!query) {
        return;
      }

      await withProgress('Searching codebase...', async () => {
        const response = await service.client!.find(query);
        if (response.count === 0) {
          void vscode.window.showInformationMessage('No matches found');
          return;
        }

        // Show results in a quick pick
        const items = response.matches.map((match) => ({
          label: `${match.file}:${match.line}`,
          description: match.reason ?? '',
          detail: match.snippet,
          file: match.file,
          line: match.line,
        }));

        const selected = await vscode.window.showQuickPick(items, {
          placeHolder: `Found ${response.count} match(es)`,
          matchOnDescription: true,
          matchOnDetail: true,
        });

        if (selected) {
          // Open the file at the line
          const uri = vscode.Uri.file(selected.file);
          const doc = await vscode.workspace.openTextDocument(uri);
          await vscode.window.showTextDocument(doc, {
            selection: new vscode.Range(selected.line - 1, 0, selected.line - 1, 0),
          });
        }
      });
    })
  );

  // Memory search command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.memorySearch', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const query = await vscode.window.showInputBox({
        prompt: 'Enter search query for similar tasks',
        placeHolder: 'What are you looking for?',
      });

      if (!query) {
        return;
      }

      await withProgress('Searching memory...', async () => {
        const response = await service.client!.memorySearch(query);
        if (response.count === 0) {
          void vscode.window.showInformationMessage('No similar tasks found');
          return;
        }

        // Show results in a quick pick
        const items = response.results.map((result) => ({
          label: result.task_id,
          description: `${Math.round(result.score * 100)}% similar`,
          detail: result.content.substring(0, 200),
        }));

        const selected = await vscode.window.showQuickPick(items, {
          placeHolder: `Found ${response.count} similar task(s)`,
          matchOnDescription: true,
          matchOnDetail: true,
        });

        if (selected) {
          void vscode.window.showInformationMessage(`Selected: ${selected.label}`);
        }
      });
    })
  );

  // Memory index command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.memoryIndex', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskId = await vscode.window.showInputBox({
        prompt: 'Enter task ID to index',
        placeHolder: 'task-1',
        value: service.currentTask?.id ?? '',
      });

      if (!taskId) {
        return;
      }

      await withProgress('Indexing task...', async () => {
        const response = await service.client!.memoryIndex(taskId);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task indexed successfully');
      });
    })
  );

  // Memory stats command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.memoryStats', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching memory stats...', async () => {
        const response = await service.client!.memoryStats();

        if (!response.enabled) {
          void vscode.window.showInformationMessage('Memory system is not enabled');
          return;
        }

        let message = `Total documents: ${response.total_documents}`;
        if (Object.keys(response.by_type).length > 0) {
          message += '\n\nBy type:';
          for (const [type, count] of Object.entries(response.by_type)) {
            message += `\n  • ${type}: ${count}`;
          }
        }

        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Library list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.libraryList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching library...', async () => {
        const response = await service.client!.libraryList();
        if (response.count === 0) {
          void vscode.window.showInformationMessage('No library collections');
          return;
        }

        const items = response.collections.map((c) => ({
          label: c.name,
          description: `${c.page_count} pages`,
          detail: `${c.source_type} - ${c.include_mode} (${c.location})`,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count} collection(s)`,
        });
      });
    })
  );

  // Library show command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.libraryShow', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter collection name or ID',
        placeHolder: 'Collection name...',
      });

      if (!name) {
        return;
      }

      await withProgress('Fetching collection...', async () => {
        const response = await service.client!.libraryShow(name);
        const c = response.collection;
        const message = `Name: ${c.name}\nSource: ${c.source}\nType: ${c.source_type}\nMode: ${c.include_mode}\nPages: ${c.page_count}\nLocation: ${c.location}`;
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Library pull command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.libraryPull', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const source = await vscode.window.showInputBox({
        prompt: 'Enter documentation source URL or path',
        placeHolder: 'https://docs.example.com or /path/to/docs',
      });

      if (!source) {
        return;
      }

      await withProgress('Pulling documentation...', async () => {
        const response = await service.client!.libraryPull(source);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Documentation pulled');
      });
    })
  );

  // Library remove command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.libraryRemove', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter collection name or ID to remove',
        placeHolder: 'Collection name...',
      });

      if (!name) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        `Remove collection "${name}"? This cannot be undone.`,
        { modal: true },
        'Remove'
      );

      if (confirm !== 'Remove') {
        return;
      }

      await withProgress('Removing collection...', async () => {
        const response = await service.client!.libraryRemove(name);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Collection removed');
      });
    })
  );

  // Library stats command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.libraryStats', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching library stats...', async () => {
        const response = await service.client!.libraryStats();

        if (!response.enabled) {
          void vscode.window.showInformationMessage('Library system is not enabled');
          return;
        }

        const message = `Collections: ${response.total_collections} (${response.shared_count} shared, ${response.project_count} project)\nPages: ${response.total_pages}\nSize: ${formatBytes(response.total_size)}`;
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Links list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.linksList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching links...', async () => {
        const response = await service.client!.linksList();
        if (response.count === 0) {
          void vscode.window.showInformationMessage('No links found');
          return;
        }

        const items = response.links.slice(0, 50).map((link) => ({
          label: `${link.source} → ${link.target}`,
          description: link.context,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count} link(s)`,
        });
      });
    })
  );

  // Links search command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.linksSearch', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const query = await vscode.window.showInputBox({
        prompt: 'Enter search query',
        placeHolder: 'Entity name...',
      });

      if (!query) {
        return;
      }

      await withProgress('Searching links...', async () => {
        const response = await service.client!.linksSearch(query);
        if (response.count === 0) {
          void vscode.window.showInformationMessage('No matching entities found');
          return;
        }

        const items = response.results.map((r) => ({
          label: r.name ?? r.entity_id,
          description: `${r.type} - ${r.entity_id}`,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count} result(s)`,
        });
      });
    })
  );

  // Links stats command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.linksStats', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching links stats...', async () => {
        const response = await service.client!.linksStats();

        if (!response.enabled) {
          void vscode.window.showInformationMessage('Links system is not enabled');
          return;
        }

        const message = `Total links: ${response.total_links}\nSources: ${response.total_sources}\nTargets: ${response.total_targets}\nOrphans: ${response.orphan_entities}`;
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Links rebuild command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.linksRebuild', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        'Rebuild the links index? This may take a moment.',
        { modal: true },
        'Rebuild'
      );

      if (confirm !== 'Rebuild') {
        return;
      }

      await withProgress('Rebuilding links index...', async () => {
        const response = await service.client!.linksRebuild();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Index rebuilt');
      });
    })
  );
}
