export function EmptyState({ title, description, icon }: { title: string; description?: string; icon?: string }) {
  return (
    <div className="flex flex-col items-center justify-center py-8 text-base-content/50">
      {icon && <span className="text-3xl mb-2">{icon}</span>}
      <p className="font-medium">{title}</p>
      {description && <p className="text-sm mt-1">{description}</p>}
    </div>
  )
}
