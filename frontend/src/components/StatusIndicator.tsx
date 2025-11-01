import { cn } from "@/lib/utils";

interface StatusIndicatorProps {
  status: 'online' | 'offline' | 'connecting';
  size?: 'sm' | 'md' | 'lg';
  showLabel?: boolean;
}

export const StatusIndicator = ({ status, size = 'md', showLabel = false }: StatusIndicatorProps) => {
  const sizeClasses = {
    sm: 'w-2 h-2',
    md: 'w-3 h-3',
    lg: 'w-4 h-4',
  };

  const statusConfig = {
    online: {
      color: 'bg-success',
      label: 'Online',
      animate: true,
      glow: 'shadow-[0_0_10px_hsl(var(--success))]',
    },
    offline: {
      color: 'bg-destructive',
      label: 'Offline',
      animate: false,
      glow: '',
    },
    connecting: {
      color: 'bg-warning',
      label: 'Connecting',
      animate: true,
      glow: 'shadow-[0_0_10px_hsl(var(--warning))]',
    },
  };

  const config = statusConfig[status];

  return (
    <div className="flex items-center gap-2">
      <div className="relative">
        <div className={cn(
          'rounded-full',
          sizeClasses[size],
          config.color,
          config.glow,
        )} />
        {config.animate && (
          <>
            <div className={cn(
              'absolute inset-0 rounded-full animate-ping',
              sizeClasses[size],
              config.color,
              'opacity-75',
            )} />
            <div className={cn(
              'absolute inset-0 rounded-full animate-pulse',
              sizeClasses[size],
              config.color,
              'opacity-40 blur-sm',
            )} />
          </>
        )}
      </div>
      {showLabel && (
        <span className="text-sm font-display font-bold uppercase tracking-wider">{config.label}</span>
      )}
    </div>
  );
};
