import * as vscode from 'vscode';
import { MehrhofProjectService } from '../services/projectService';
import type { StateChangedEvent, AgentMessageEvent } from '../api/models';

export class MehrhofOutputChannel implements vscode.Disposable {
  private readonly channel: vscode.OutputChannel;
  private readonly service: MehrhofProjectService;

  constructor(service: MehrhofProjectService) {
    this.service = service;
    this.channel = vscode.window.createOutputChannel('Mehrhof');

    // Listen for events
    this.service.on('connectionChanged', (state) => {
      this.log(`Connection: ${state}`);
    });

    this.service.on('stateChanged', (event: StateChangedEvent) => {
      this.log(`--- State changed: ${event.from} → ${event.to} ---`);
    });

    this.service.on('agentMessage', (event: AgentMessageEvent) => {
      this.logAgentMessage(event);
    });

    this.service.on('taskChanged', (task, work) => {
      if (task) {
        this.log(`Task: ${work?.title ?? task.id} (${task.state})`);
      }
    });

    this.service.on('questionReceived', (question) => {
      this.log(`Question: ${question.question}`);
      if (question.options?.length) {
        this.log(`  Options: ${question.options.join(', ')}`);
      }
    });

    this.service.on('error', (error) => {
      this.log(`Error: ${error.message}`);
    });
  }

  get outputChannel(): vscode.OutputChannel {
    return this.channel;
  }

  show(): void {
    this.channel.show();
  }

  clear(): void {
    this.channel.clear();
  }

  log(message: string): void {
    const timestamp = new Date().toISOString().split('T')[1].split('.')[0];
    this.channel.appendLine(`[${timestamp}] ${message}`);
  }

  private logAgentMessage(event: AgentMessageEvent): void {
    const rolePrefix = this.getRolePrefix(event.role);
    const lines = event.content.split('\n');
    for (const line of lines) {
      this.channel.appendLine(`${rolePrefix} ${line}`);
    }
  }

  private getRolePrefix(role: string): string {
    switch (role) {
      case 'assistant':
        return '[Agent]';
      case 'tool':
        return '[Tool]';
      case 'system':
        return '[System]';
      default:
        return `[${role}]`;
    }
  }

  dispose(): void {
    this.channel.dispose();
  }
}
