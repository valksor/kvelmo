import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import { ApiError } from '../api/client';

export function registerCommands(
  context: vscode.ExtensionContext,
  service: MehrhofProjectService
): void {
  // Server commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.startServer', async () => {
      await withProgress('Starting server...', async () => {
        await service.startServer();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stopServer', () => {
      service.stopServer();
      void vscode.window.showInformationMessage('Mehrhof: Server stopped');
    })
  );

  // Connection commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.connect', async () => {
      await withProgress('Connecting...', async () => {
        await service.connect();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.disconnect', () => {
      service.disconnect();
      void vscode.window.showInformationMessage('Mehrhof: Disconnected');
    })
  );

  // Workflow commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.startTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const ref = await vscode.window.showInputBox({
        prompt: 'Enter task reference (e.g., github:123, file:path/to/task.md)',
        placeHolder: 'Task reference',
      });

      if (!ref) {
        return;
      }

      await withProgress('Starting task...', async () => {
        const response = await service.client!.startTask({ ref });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.plan', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Planning...', async () => {
        const response = await service.client!.plan();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.implement', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Implementing...', async () => {
        const response = await service.client!.implement();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.review', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Reviewing...', async () => {
        const response = await service.client!.review();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.continue', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Continuing...', async () => {
        const response = await service.client!.continueWorkflow();
        if (!response.success) {
          throw new Error('Continue failed');
        }
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.finish', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showQuickPick(['Yes', 'No'], {
        placeHolder: 'Finish the current task?',
      });

      if (confirm !== 'Yes') {
        return;
      }

      await withProgress('Finishing...', async () => {
        const response = await service.client!.finish();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.abandon', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        'Are you sure you want to abandon the current task? This cannot be undone.',
        { modal: true },
        'Abandon'
      );

      if (confirm !== 'Abandon') {
        return;
      }

      await withProgress('Abandoning...', async () => {
        const response = await service.client!.abandon();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  // Checkpoint commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.undo', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Undoing...', async () => {
        const response = await service.client!.undo();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.redo', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Redoing...', async () => {
        const response = await service.client!.redo();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
      });
    })
  );

  // Info commands
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.status', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await service.refreshState();

      const task = service.currentTask;
      const work = service.currentWork;
      const state = service.workflowState;

      if (task) {
        const message = `Task: ${work?.title ?? task.id}\nState: ${state}\nBranch: ${task.branch ?? 'N/A'}`;
        void vscode.window.showInformationMessage(message, { modal: true });
      } else {
        void vscode.window.showInformationMessage('No active task');
      }
    })
  );

  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.refresh', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await service.refreshState();
      void vscode.window.showInformationMessage('Mehrhof: Refreshed');
    })
  );

  // Note command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.note', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        void vscode.window.showWarningMessage('Mehrhof: No active task');
        return;
      }

      const message = await vscode.window.showInputBox({
        prompt: 'Enter note message',
        placeHolder: 'Note content...',
      });

      if (!message) {
        return;
      }

      await withProgress('Adding note...', async () => {
        const response = await service.client!.addNote(task.id, { message });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(
          `Mehrhof: Note #${response.note_number ?? 'N/A'} added`
        );
      });
    })
  );

  // Question command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.question', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        void vscode.window.showWarningMessage('Mehrhof: No active task');
        return;
      }

      const message = await vscode.window.showInputBox({
        prompt: 'Enter question for the agent',
        placeHolder: 'Your question...',
      });

      if (!message) {
        return;
      }

      await withProgress('Asking question...', async () => {
        const response = await service.client!.question({ message });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
      });
    })
  );

  // Reset command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.reset', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        'Reset workflow state to idle? This will not lose your work.',
        { modal: true },
        'Reset'
      );

      if (confirm !== 'Reset') {
        return;
      }

      await withProgress('Resetting...', async () => {
        const response = await service.client!.reset();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        await service.refreshState();
        void vscode.window.showInformationMessage('Mehrhof: Workflow reset to idle');
      });
    })
  );

  // Cost command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.cost', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        // Show all costs if no active task
        await withProgress('Fetching costs...', async () => {
          const response = await service.client!.getAllCosts();
          const total = response.grand_total;
          const message = `Total Cost: $${total.cost_usd.toFixed(4)}\nTokens: ${total.total_tokens.toLocaleString()} (${total.input_tokens.toLocaleString()} in, ${total.output_tokens.toLocaleString()} out)\nCached: ${total.cached_tokens.toLocaleString()}`;
          void vscode.window.showInformationMessage(message, { modal: true });
        });
        return;
      }

      await withProgress('Fetching task costs...', async () => {
        const response = await service.client!.getTaskCosts(task.id);
        const message = `Task: ${response.title ?? task.id}\nCost: $${response.total_cost_usd.toFixed(4)}\nTokens: ${response.total_tokens.toLocaleString()} (${response.input_tokens.toLocaleString()} in, ${response.output_tokens.toLocaleString()} out)\nCached: ${response.cached_tokens.toLocaleString()} (${response.cached_percent?.toFixed(1) ?? 0}%)`;
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Quick task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.quick', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const description = await vscode.window.showInputBox({
        prompt: 'Enter task description',
        placeHolder: 'Task description...',
      });

      if (!description) {
        return;
      }

      await withProgress('Creating quick task...', async () => {
        const response = await service.client!.executeCommand({
          command: 'quick',
          args: [description],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Quick task created');
      });
    })
  );

  // Delete queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.deleteTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const confirm = await vscode.window.showWarningMessage(
        `Delete task ${taskRef}? This cannot be undone.`,
        { modal: true },
        'Delete'
      );

      if (confirm !== 'Delete') {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Deleting task...', async () => {
        const response = await service.client!.deleteQueueTask(queueId, taskId);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task deleted');
      });
    })
  );

  // Export queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.exportTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Exporting task...', async () => {
        const response = await service.client!.exportQueueTask(queueId, taskId);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        if (response.markdown) {
          // Open the markdown in a new document
          const doc = await vscode.workspace.openTextDocument({
            content: response.markdown,
            language: 'markdown',
          });
          await vscode.window.showTextDocument(doc);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task exported');
      });
    })
  );

  // Optimize queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.optimizeTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Optimizing task with AI...', async () => {
        const response = await service.client!.optimizeQueueTask(queueId, taskId);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        let message = 'Task optimized';
        if (response.original_title !== response.optimized_title) {
          message += `\nTitle: ${response.original_title} → ${response.optimized_title}`;
        }
        if (response.added_labels && response.added_labels.length > 0) {
          message += `\nAdded labels: ${response.added_labels.join(', ')}`;
        }
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Submit queue task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.submitTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskRef = await vscode.window.showInputBox({
        prompt: 'Enter task reference (queue/task-id)',
        placeHolder: 'quick-tasks/task-1',
      });

      if (!taskRef) {
        return;
      }

      const provider = await vscode.window.showInputBox({
        prompt: 'Enter provider name',
        placeHolder: 'github, jira, wrike, linear, etc.',
      });

      if (!provider) {
        return;
      }

      const [queueId, taskId] = taskRef.split('/');
      if (!queueId || !taskId) {
        void vscode.window.showErrorMessage('Invalid task reference format');
        return;
      }

      await withProgress('Submitting task...', async () => {
        const response = await service.client!.submitQueueTask(queueId, taskId, provider);
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        let message = response.message ?? 'Task submitted';
        if (response.external_id) {
          message += `\nExternal ID: ${response.external_id}`;
        }
        if (response.url) {
          message += `\nURL: ${response.url}`;
        }
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Sync task command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.syncTask', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const task = service.currentTask;
      if (!task) {
        void vscode.window.showWarningMessage('Mehrhof: No active task to sync');
        return;
      }

      await withProgress('Syncing task...', async () => {
        const response = await service.client!.syncTask();
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task synced');
      });
    })
  );

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

  // Browser status command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserStatus', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Checking browser status...', async () => {
        const response = await service.client!.browserStatus();
        if (!response.connected) {
          void vscode.window.showInformationMessage(
            `Browser: Not connected${response.error ? ` (${response.error})` : ''}`
          );
          return;
        }
        const message = `Browser: Connected to ${response.host}:${response.port}\nTabs: ${response.tabs?.length ?? 0}`;
        void vscode.window.showInformationMessage(message, { modal: true });
      });
    })
  );

  // Browser tabs command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserTabs', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching browser tabs...', async () => {
        const response = await service.client!.browserTabs();
        if (response.count === 0) {
          void vscode.window.showInformationMessage('No browser tabs open');
          return;
        }

        const items = response.tabs.map((tab) => ({
          label: tab.title || 'Untitled',
          description: truncateUrl(tab.url, 60),
          detail: tab.id,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count} tab(s)`,
        });
      });
    })
  );

  // Browser go to command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserGoto', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const url = await vscode.window.showInputBox({
        prompt: 'Enter URL to navigate to',
        placeHolder: 'https://example.com',
      });

      if (!url) {
        return;
      }

      await withProgress('Opening URL...', async () => {
        const response = await service.client!.browserGoto({ url });
        if (response.success && response.tab) {
          void vscode.window.showInformationMessage(
            `Opened: ${response.tab.title || response.tab.url}`
          );
        }
      });
    })
  );

  // Browser navigate command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserNavigate', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const url = await vscode.window.showInputBox({
        prompt: 'Enter URL to navigate current tab to',
        placeHolder: 'https://example.com',
      });

      if (!url) {
        return;
      }

      await withProgress('Navigating...', async () => {
        const response = await service.client!.browserNavigate({ url });
        if (!response.success) {
          throw new Error('Navigation failed');
        }
        void vscode.window.showInformationMessage(response.message ?? 'Navigated');
      });
    })
  );

  // Browser reload command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserReload', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Reloading page...', async () => {
        const response = await service.client!.browserReload({});
        if (!response.success) {
          throw new Error('Reload failed');
        }
        void vscode.window.showInformationMessage(response.message ?? 'Page reloaded');
      });
    })
  );

  // Browser screenshot command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserScreenshot', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Taking screenshot...', async () => {
        const response = await service.client!.browserScreenshot({});
        if (!response.success || !response.data) {
          throw new Error('Screenshot failed');
        }

        // Show base64 image info
        const sizeKb = response.size ? Math.round(response.size / 1024) : 0;
        void vscode.window.showInformationMessage(
          `Screenshot captured: ${response.format ?? 'png'}, ${sizeKb} KB`
        );
      });
    })
  );

  // Browser click command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserClick', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const selector = await vscode.window.showInputBox({
        prompt: 'Enter CSS selector to click',
        placeHolder: '#button, .submit-btn, button[type="submit"]',
      });

      if (!selector) {
        return;
      }

      await withProgress('Clicking element...', async () => {
        const response = await service.client!.browserClick({ selector });
        if (!response.success) {
          throw new Error('Click failed');
        }
        void vscode.window.showInformationMessage(`Clicked: ${response.selector ?? selector}`);
      });
    })
  );

  // Browser type command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserType', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const selector = await vscode.window.showInputBox({
        prompt: 'Enter CSS selector for input element',
        placeHolder: '#search, input[name="query"]',
      });

      if (!selector) {
        return;
      }

      const text = await vscode.window.showInputBox({
        prompt: 'Enter text to type',
        placeHolder: 'Text to type...',
      });

      if (text === undefined) {
        return;
      }

      await withProgress('Typing...', async () => {
        const response = await service.client!.browserType({ selector, text });
        if (!response.success) {
          throw new Error('Type failed');
        }
        void vscode.window.showInformationMessage(`Typed into: ${response.selector ?? selector}`);
      });
    })
  );

  // Browser eval command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserEval', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const expression = await vscode.window.showInputBox({
        prompt: 'Enter JavaScript expression to evaluate',
        placeHolder: 'document.title',
      });

      if (!expression) {
        return;
      }

      await withProgress('Evaluating...', async () => {
        const response = await service.client!.browserEval({ expression });
        if (!response.success) {
          throw new Error('Evaluation failed');
        }
        const resultStr = JSON.stringify(response.result, null, 2);
        void vscode.window.showInformationMessage(`Result: ${resultStr}`, { modal: true });
      });
    })
  );

  // Browser console command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserConsole', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching console logs...', async () => {
        const response = await service.client!.browserConsole({});
        if (!response.messages || response.messages.length === 0) {
          void vscode.window.showInformationMessage('No console messages');
          return;
        }

        const items = response.messages.map((msg) => ({
          label: `[${msg.level.toUpperCase()}]`,
          description: msg.text.substring(0, 100),
          detail: msg.timestamp,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count ?? response.messages.length} message(s)`,
        });
      });
    })
  );

  // Browser network command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.browserNetwork', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching network requests...', async () => {
        const response = await service.client!.browserNetwork({});
        if (!response.requests || response.requests.length === 0) {
          void vscode.window.showInformationMessage('No network requests');
          return;
        }

        const items = response.requests.map((req) => ({
          label: `${req.method} ${req.status ?? '...'}`,
          description: truncateUrl(req.url, 60),
          detail: req.timestamp,
        }));

        await vscode.window.showQuickPick(items, {
          placeHolder: `${response.count ?? response.requests.length} request(s)`,
        });
      });
    })
  );

  // ============================================================================
  // Project Commands
  // ============================================================================

  // Project plan command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectPlan', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const source = await vscode.window.showInputBox({
        prompt: 'Enter source (file path, URL, or GitHub issue reference)',
        placeHolder: 'e.g., ./roadmap.md or owner/repo#123',
      });

      if (!source) {
        return;
      }

      await withProgress('Creating project plan...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['plan', source],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Project plan created');
      });
    })
  );

  // Project tasks command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectTasks', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching project tasks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['tasks'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Project tasks retrieved');
      });
    })
  );

  // Project edit command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectEdit', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskId = await vscode.window.showInputBox({
        prompt: 'Enter task ID to edit',
        placeHolder: 'e.g., task-1',
      });

      if (!taskId) {
        return;
      }

      const field = await vscode.window.showQuickPick(
        [
          { label: 'title', description: 'Edit task title' },
          { label: 'priority', description: 'Edit task priority' },
          { label: 'status', description: 'Edit task status' },
        ],
        { placeHolder: 'Select field to edit' }
      );

      if (!field) {
        return;
      }

      const value = await vscode.window.showInputBox({
        prompt: `Enter new ${field.label}`,
        placeHolder: `New ${field.label} value`,
      });

      if (!value) {
        return;
      }

      await withProgress('Updating project task...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['edit', taskId, `--${field.label}`, value],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Task updated');
      });
    })
  );

  // Project submit command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectSubmit', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const provider = await vscode.window.showQuickPick(
        [
          { label: 'github', description: 'Submit to GitHub Issues' },
          { label: 'gitlab', description: 'Submit to GitLab Issues' },
          { label: 'linear', description: 'Submit to Linear' },
          { label: 'jira', description: 'Submit to Jira' },
        ],
        { placeHolder: 'Select provider' }
      );

      if (!provider) {
        return;
      }

      await withProgress('Submitting project tasks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['submit', provider.label],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Tasks submitted');
      });
    })
  );

  // Project start command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectStart', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Starting next project task...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['start'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Started next task');
      });
    })
  );

  // Project sync command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.projectSync', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const reference = await vscode.window.showInputBox({
        prompt: 'Enter provider reference to sync from',
        placeHolder: 'e.g., owner/repo#123 or PROJECT-123',
      });

      if (!reference) {
        return;
      }

      await withProgress('Syncing project...', async () => {
        const response = await service.client!.executeCommand({
          command: 'project',
          args: ['sync', reference],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Project synced');
      });
    })
  );

  // ============================================================================
  // Stack Commands
  // ============================================================================

  // Stack list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stackList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Fetching stacks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'stack',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Stacks retrieved');
      });
    })
  );

  // Stack rebase command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stackRebase', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const taskId = await vscode.window.showInputBox({
        prompt: 'Enter task ID to rebase (leave empty to rebase all)',
        placeHolder: 'e.g., task-1 (optional)',
      });

      await withProgress('Rebasing stack...', async () => {
        const args = taskId ? ['rebase', taskId] : ['rebase'];
        const response = await service.client!.executeCommand({
          command: 'stack',
          args,
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Stack rebased');
      });
    })
  );

  // Stack sync command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.stackSync', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Syncing stacks...', async () => {
        const response = await service.client!.executeCommand({
          command: 'stack',
          args: ['sync'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Stacks synced');
      });
    })
  );

  // ============================================================================
  // Configuration Commands
  // ============================================================================

  // Config validate command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.configValidate', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Validating configuration...', async () => {
        const response = await service.client!.executeCommand({
          command: 'config',
          args: ['validate'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Configuration valid');
      });
    })
  );

  // Agents list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.agentsList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Listing agents...', async () => {
        const response = await service.client!.executeCommand({
          command: 'agents',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Agents retrieved');
      });
    })
  );

  // Agents explain command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.agentsExplain', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter agent name to explain',
        placeHolder: 'e.g., claude',
      });

      if (!name) {
        return;
      }

      await withProgress('Getting agent info...', async () => {
        const response = await service.client!.executeCommand({
          command: 'agents',
          args: ['explain', name],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Agent info retrieved');
      });
    })
  );

  // Providers list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.providersList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Listing providers...', async () => {
        const response = await service.client!.executeCommand({
          command: 'providers',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Providers retrieved');
      });
    })
  );

  // Providers info command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.providersInfo', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter provider name',
        placeHolder: 'e.g., github, jira, linear',
      });

      if (!name) {
        return;
      }

      await withProgress('Getting provider info...', async () => {
        const response = await service.client!.executeCommand({
          command: 'providers',
          args: ['info', name],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Provider info retrieved');
      });
    })
  );

  // Templates list command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.templatesList', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Listing templates...', async () => {
        const response = await service.client!.executeCommand({
          command: 'templates',
          args: ['list'],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Templates retrieved');
      });
    })
  );

  // Templates show command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.templatesShow', async () => {
      if (!requireConnection(service)) {
        return;
      }

      const name = await vscode.window.showInputBox({
        prompt: 'Enter template name',
        placeHolder: 'e.g., bug-fix, feature, refactor',
      });

      if (!name) {
        return;
      }

      await withProgress('Getting template...', async () => {
        const response = await service.client!.executeCommand({
          command: 'templates',
          args: ['show', name],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Template retrieved');
      });
    })
  );

  // Scan command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.scan', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Running security scan...', async () => {
        const response = await service.client!.executeCommand({
          command: 'scan',
          args: [],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Scan complete');
      });
    })
  );

  // Commit command
  context.subscriptions.push(
    vscode.commands.registerCommand('mehrhof.commit', async () => {
      if (!requireConnection(service)) {
        return;
      }

      await withProgress('Running commit analysis...', async () => {
        const response = await service.client!.executeCommand({
          command: 'commit',
          args: [],
        });
        if (!response.success && response.error) {
          throw new Error(response.error);
        }
        void vscode.window.showInformationMessage(response.message ?? 'Commit analysis complete');
      });
    })
  );
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

function truncateUrl(url: string, maxLen: number): string {
  if (url.length <= maxLen) return url;
  return url.substring(0, maxLen - 3) + '...';
}

function requireConnection(service: MehrhofProjectService): boolean {
  if (!service.isConnected || !service.client) {
    void vscode.window.showWarningMessage('Mehrhof: Not connected. Please connect first.');
    return false;
  }
  return true;
}

async function withProgress<T>(title: string, task: () => Promise<T>): Promise<T | undefined> {
  try {
    return await vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: `Mehrhof: ${title}`,
        cancellable: false,
      },
      async () => {
        return await task();
      }
    );
  } catch (error) {
    const message =
      error instanceof ApiError
        ? error.message
        : error instanceof Error
          ? error.message
          : 'Unknown error';
    void vscode.window.showErrorMessage(`Mehrhof: ${message}`);
    return undefined;
  }
}
