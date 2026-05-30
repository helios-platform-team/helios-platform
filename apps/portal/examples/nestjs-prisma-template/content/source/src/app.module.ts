import { Module } from '@nestjs/common';
import { AppController } from './app.controller';
import { AppService } from './app.service';
{% if values.hasDatabase -%}
import { PrismaModule } from './prisma/prisma.module';
{% endif -%}

@Module({
  imports: [
{% if values.hasDatabase -%}
    PrismaModule,
{% endif -%}
  ],
  controllers: [AppController],
  providers: [AppService],
})
export class AppModule {}
