import { useState } from 'react'
import { Send, Loader2 } from 'lucide-react'
import { useAddNote } from '@/api/task'
import type { Note } from '@/types/api'

interface NotesCardProps {
  notes?: Note[]
  taskId?: string
}

export function NotesCard({ notes, taskId }: NotesCardProps) {
  const [newNote, setNewNote] = useState('')
  const { mutate: addNote, isPending } = useAddNote(taskId)

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!newNote.trim() || !taskId) return

    addNote(newNote.trim(), {
      onSuccess: () => setNewNote(''),
    })
  }

  return (
    <div className="card bg-base-100 shadow-sm">
      <div className="card-body">
        {/* Header */}
        <div className="flex items-center justify-between pb-4 border-b border-base-200">
          <h3 className="text-lg font-bold text-base-content">
            Notes {notes && notes.length > 0 && `(${notes.length})`}
          </h3>
          <span className="text-xs text-base-content/60">User input and Q&A history</span>
        </div>

        {/* Add note form */}
        <form onSubmit={handleSubmit} className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end">
          <div className="form-control flex-1">
            <label className="label py-1" htmlFor="task-note-input">
              <span className="label-text">New note</span>
            </label>
            <input
              id="task-note-input"
              type="text"
              value={newNote}
              onChange={(e) => setNewNote(e.target.value)}
              placeholder="Add a note..."
              className="input input-bordered w-full"
              disabled={isPending || !taskId}
            />
          </div>
          <button
            type="submit"
            className="btn btn-primary"
            disabled={isPending || !newNote.trim() || !taskId}
            aria-label="Send note"
          >
            {isPending ? <Loader2 size={16} className="animate-spin" aria-hidden="true" /> : <Send size={16} aria-hidden="true" />}
          </button>
        </form>

        {/* Notes list */}
        {notes && notes.length > 0 ? (
          <div className="divide-y divide-base-200 mt-4 max-h-96 overflow-y-auto">
            {notes.map((note) => (
              <NoteItem key={note.number} note={note} />
            ))}
          </div>
        ) : (
          <p className="text-base-content/60 text-center py-8">No notes yet</p>
        )}
      </div>
    </div>
  )
}

interface NoteItemProps {
  note: Note
}

function NoteItem({ note }: NoteItemProps) {
  return (
    <div className="py-4">
      <div className="flex items-center gap-2 text-xs text-base-content/60 mb-2">
        <span className="font-mono bg-base-200 px-1.5 py-0.5 rounded">#{note.number}</span>
        <span>{note.timestamp}</span>
        {note.state && (
          <span className="px-1.5 py-0.5 rounded bg-primary/10 text-primary text-xs">
            {note.state}
          </span>
        )}
      </div>
      <div className="text-sm text-base-content/80 whitespace-pre-wrap">{note.content}</div>
    </div>
  )
}
