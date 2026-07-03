import { Injectable } from '@nestjs/common';
{% if values.hasDatabase %}import { PrismaService } from './prisma/prisma.service';{% endif %}

@Injectable()
export class AppService {
  {% if values.hasDatabase %}constructor(private readonly prisma: PrismaService) {}{% endif %}

  getHello(): string {
    return 'Hello from ${{ values.name }}!';
  }

  async healthCheck(): Promise<{ status: string; database?: string }> {
{% if values.hasDatabase %}    try {
      await this.prisma.$queryRaw`SELECT 1`;
      return { status: 'ok', database: 'connected' };
    } catch (err: any) {
      return { status: 'error', database: `disconnected: ${err.message}` };
    }{% else %}    return { status: 'ok' };{% endif %}
  }
}
