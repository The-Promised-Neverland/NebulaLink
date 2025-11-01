import { Card } from "@/components/ui/card";
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from 'recharts';

interface MetricsChartProps {
  data: Array<{
    timestamp: string;
    cpu: number;
    memory: number;
    disk: number;
  }>;
}

export const MetricsChart = ({ data }: MetricsChartProps) => {
  return (
    <Card className="relative p-6 glass-effect border-primary/20 overflow-hidden group">
      {/* Animated background gradient */}
      <div className="absolute inset-0 bg-gradient-to-br from-primary/5 via-transparent to-secondary/5 opacity-50" />
      
      {/* Corner decorations */}
      <div className="absolute top-0 left-0 w-16 h-16 border-t-2 border-l-2 border-primary/30" />
      <div className="absolute top-0 right-0 w-16 h-16 border-t-2 border-r-2 border-primary/30" />
      <div className="absolute bottom-0 left-0 w-16 h-16 border-b-2 border-l-2 border-primary/30" />
      <div className="absolute bottom-0 right-0 w-16 h-16 border-b-2 border-r-2 border-primary/30" />
      
      <h3 className="relative text-xl font-display font-bold mb-6 text-foreground text-glow uppercase tracking-wider">System Metrics</h3>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data}>
          <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border) / 0.3)" />
          <XAxis
            dataKey="timestamp"
            stroke="hsl(var(--primary))"
            tick={{ fill: 'hsl(var(--muted-foreground))', fontSize: 12 }}
            style={{ fontFamily: 'Share Tech Mono, monospace' }}
          />
          <YAxis
            stroke="hsl(var(--primary))"
            tick={{ fill: 'hsl(var(--muted-foreground))', fontSize: 12 }}
            domain={[0, 100]}
            style={{ fontFamily: 'Share Tech Mono, monospace' }}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: 'hsl(var(--popover) / 0.95)',
              backdropFilter: 'blur(12px)',
              border: '1px solid hsl(var(--primary) / 0.3)',
              borderRadius: '0.5rem',
              boxShadow: '0 0 20px hsl(var(--primary) / 0.2)',
            }}
            labelStyle={{ 
              color: 'hsl(var(--foreground))',
              fontFamily: 'Orbitron, sans-serif',
              fontWeight: 'bold',
            }}
            itemStyle={{
              fontFamily: 'Share Tech Mono, monospace',
            }}
          />
          <Legend
            wrapperStyle={{
              color: 'hsl(var(--foreground))',
              fontFamily: 'Orbitron, sans-serif',
              fontWeight: 'bold',
            }}
          />
          <Line
            type="monotone"
            dataKey="cpu"
            stroke="hsl(var(--chart-1))"
            strokeWidth={3}
            dot={{ fill: 'hsl(var(--chart-1))', r: 4, strokeWidth: 2, stroke: 'hsl(var(--background))' }}
            activeDot={{ r: 6, fill: 'hsl(var(--chart-1))', stroke: 'hsl(var(--primary-bright))', strokeWidth: 2 }}
            name="CPU %"
          />
          <Line
            type="monotone"
            dataKey="memory"
            stroke="hsl(var(--chart-2))"
            strokeWidth={3}
            dot={{ fill: 'hsl(var(--chart-2))', r: 4, strokeWidth: 2, stroke: 'hsl(var(--background))' }}
            activeDot={{ r: 6, fill: 'hsl(var(--chart-2))', stroke: 'hsl(var(--chart-2))', strokeWidth: 2 }}
            name="Memory %"
          />
          <Line
            type="monotone"
            dataKey="disk"
            stroke="hsl(var(--chart-3))"
            strokeWidth={3}
            dot={{ fill: 'hsl(var(--chart-3))', r: 4, strokeWidth: 2, stroke: 'hsl(var(--background))' }}
            activeDot={{ r: 6, fill: 'hsl(var(--chart-3))', stroke: 'hsl(var(--chart-3))', strokeWidth: 2 }}
            name="Disk %"
          />
        </LineChart>
      </ResponsiveContainer>
    </Card>
  );
};
