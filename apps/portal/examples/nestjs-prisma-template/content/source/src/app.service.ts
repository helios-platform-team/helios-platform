import { Injectable } from '@nestjs/common';
{% if values.hasDatabase -%}
import { PrismaService } from './prisma/prisma.service';
{% endif -%}

@Injectable()
export class AppService {
{% if values.hasDatabase -%}
  constructor(private readonly prisma: PrismaService) {}
{% endif -%}

  getHello(): string {
    return 'Hello from ${{ values.name }}!';
  }

  healthCheck(): { status: string } {
    return { status: 'ok' };
  }
}
